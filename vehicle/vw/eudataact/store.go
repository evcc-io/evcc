package eudataact

import (
	"slices"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// maxBackfill bounds how many of the most recent content datasets are downloaded
// on the first poll of a session to seed the merged map.
const maxBackfill = 8

// store holds the merged dataset state for all vehicles of a single portal
// account consisting of username and brand
type store struct {
	mu       sync.Mutex // guards vehicles
	api      *API
	vehicles map[string]*vehicleState
}

// vehicleState holds the merged dataset state for a single vehicle
type vehicleState struct {
	mu         sync.Mutex // guards the fields below
	identifier string
	data       []point
	after      time.Time
	seq        uint64 // delivery counter, incremented per merged dataset
}

var (
	storeMu  sync.Mutex
	storeReg = make(map[*API]*store)
)

// sharedStore returns the store shared by all vehicles of the given account,
// creating it on first use. Since NewAPI already returns one client per username
// and brand, keying on the client yields a single store per account.
func sharedStore(api *API) *store {
	storeMu.Lock()
	defer storeMu.Unlock()

	if s, ok := storeReg[api]; ok {
		return s
	}

	s := &store{
		api:      api,
		vehicles: make(map[string]*vehicleState),
	}
	storeReg[api] = s

	return s
}

// state returns the per-vehicle state for vin, creating it on first use
func (s *store) state(vin string) *vehicleState {
	s.mu.Lock()
	defer s.mu.Unlock()

	v := s.vehicles[vin]
	if v == nil {
		v = &vehicleState{}
		s.vehicles[vin] = v
	}

	return v
}

// update downloads any datasets for vin delivered after the newest one already
// merged and merges them into the vehicle's map oldest to newest. It returns the
// newest dataset's delivery time (used to schedule the next poll).
// On first poll latest maxBackfill content datasets are downloaded.
func (s *store) update(vin string) (time.Time, error) {
	v := s.state(vin)

	v.mu.Lock()
	defer v.mu.Unlock()

	if v.identifier == "" {
		id, err := s.api.identifier(vin)
		if err != nil {
			return time.Time{}, err
		}
		v.identifier = id
	}

	list, err := s.api.datasets(vin, v.identifier)
	if err != nil {
		return time.Time{}, err
	}

	content := contentDatasets(list)

	var newest time.Time
	for _, d := range list {
		if d.CreatedOn.After(newest) {
			newest = d.CreatedOn
		}
	}

	// on the first poll the backfilled datasets are logged once as the final
	// merged map below; afterwards each newly received dataset is logged as it
	// arrives
	initial := v.after.IsZero()

	for _, d := range pending(content, v.after) {
		b, err := s.api.download(vin, v.identifier, d.Name)
		if err != nil {
			return newest, err
		}

		data, err := parseDataset(b)
		if err != nil {
			return newest, err
		}

		// advance the high-water mark so this dataset is never downloaded again,
		// even when it is dropped below
		if d.CreatedOn.After(v.after) {
			v.after = d.CreatedOn
		}

		v.seq++
		v.data = merge(v.data, data, v.seq)

		if !initial {
			logData(s.api.log, data)
		}
	}

	if len(v.data) == 0 {
		return time.Time{}, api.ErrNotAvailable
	}

	if initial {
		logData(s.api.log, v.data)
	}

	return newest, nil
}

// snapshot returns a copy of the merged data for vin
func (s *store) snapshot(vin string) []point {
	v := s.state(vin)

	v.mu.Lock()
	defer v.mu.Unlock()

	return slices.Clone(v.data)
}

// logData logs every data point at DEBUG level in arrival order, with its value
// and own timestamp in local time.
func logData(log *util.Logger, data []point) {
	for _, p := range data {
		log.DEBUG.Printf("recv %s %s: %s (%s)", p.Key, p.Name, p.Value, p.Timestamp.Local().Format("2006-01-02 15:04:05"))
	}
}

// pending returns the content datasets that still need downloading, oldest to
// newest. content must be sorted oldest to newest. On the first poll (after is
// zero) only the latest maxBackfill datasets are returned to seed the map;
// afterwards only datasets delivered after the newest one already merged are
// returned, so no dataset is downloaded twice.
func pending(content []dataset, after time.Time) []dataset {
	if after.IsZero() {
		if len(content) > maxBackfill {
			return content[len(content)-maxBackfill:]
		}
		return content
	}

	res := make([]dataset, 0, len(content))
	for _, d := range content {
		if d.CreatedOn.After(after) {
			res = append(res, d)
		}
	}

	return res
}

// merge folds src (the newer dataset) into dst per id, stamping each point with
// seq, the dataset's delivery sequence (timestampUtc is unreliable).
func merge(dst, src []point, seq uint64) []point {
	for _, p := range src {
		p.Seq = seq
		if e := find(dst, p.id()); e != nil {
			*e = p
		} else {
			dst = append(dst, p)
		}
	}
	return dst
}
