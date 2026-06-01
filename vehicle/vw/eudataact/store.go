package eudataact

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// maxBackfill bounds how many of the most recent content datasets are downloaded
// on the first poll of a session to seed the merged map.
const maxBackfill = 8

// store holds the merged dataset state for a single vehicle across a session.
// The portal only ever appends datasets. Each dataset is downloaded at most once;
// its data points are merged into data, keeping the newest value per field. after
// is the delivery time of the newest dataset already merged, so later polls only
// fetch datasets delivered after it and nothing is downloaded twice.
type store struct {
	mu         sync.Mutex
	api        *API
	vin        string
	identifier string
	data       map[string]point
	after      time.Time
}

// newStore creates an empty store for the given vehicle
func newStore(api *API, vin string) *store {
	return &store{
		api:  api,
		vin:  vin,
		data: make(map[string]point),
	}
}

// update downloads any datasets delivered after the newest one already merged
// and merges them into the map oldest to newest. It returns the newest dataset's
// delivery time (used to schedule the next poll); the merged data is read with
// snapshot. On the first poll only the latest maxBackfill content datasets are
// downloaded.
func (s *store) update() (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.identifier == "" {
		id, err := s.api.identifier(s.vin)
		if err != nil {
			return time.Time{}, err
		}
		s.identifier = id
	}

	list, err := s.api.datasets(s.vin, s.identifier)
	if err != nil {
		return time.Time{}, err
	}

	content, err := contentDatasets(list)
	if err != nil {
		return time.Time{}, err
	}

	var newest time.Time
	for _, d := range list {
		t, err := d.time()
		if err != nil {
			return time.Time{}, err
		}
		if t.After(newest) {
			newest = t
		}
	}

	// on the first poll the backfilled datasets are logged once as the final
	// merged map below; afterwards each newly received dataset is logged as it
	// arrives
	initial := s.after.IsZero()

	for _, d := range pending(content, s.after) {
		b, err := s.api.download(s.vin, s.identifier, d.Name)
		if err != nil {
			return newest, err
		}

		data, err := parseDataset(b)
		if err != nil {
			return newest, err
		}

		merge(s.data, data)

		// advance the high-water mark so this dataset is never downloaded again
		if d.Timestamp.After(s.after) {
			s.after = d.Timestamp
		}

		if !initial {
			logData(s.api.log, data)
		}
	}

	if len(s.data) == 0 {
		return time.Time{}, api.ErrNotAvailable
	}

	if initial {
		logData(s.api.log, s.data)
	}

	return newest, nil
}

// snapshot returns a copy of the merged data
func (s *store) snapshot() map[string]point {
	s.mu.Lock()
	defer s.mu.Unlock()

	return maps.Clone(s.data)
}

// logData logs every field of data at DEBUG level, sorted by field name, with
// its value and own timestamp in local time.
func logData(log *util.Logger, data map[string]point) {
	for _, k := range slices.Sorted(maps.Keys(data)) {
		p := data[k]
		log.DEBUG.Printf("recv %s: %s (%s)", k, p.Value, p.Timestamp.Local().Format("2006-01-02 15:04:05"))
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
		if d.Timestamp.After(after) {
			res = append(res, d)
		}
	}

	return res
}

// merge copies the data points from src into dst, keeping the newest value per
// field across datasets.
func merge(dst, src map[string]point) {
	for k, p := range src {
		if cur, ok := dst[k]; ok && cur.Timestamp.After(p.Timestamp) {
			continue
		}
		dst[k] = p
	}
}
