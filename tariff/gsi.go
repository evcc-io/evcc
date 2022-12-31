package tariff

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type GrünStromIndex struct {
	*request.Helper
	log     *util.Logger
	mux     sync.Mutex
	zip     string
	data    api.Rates
	updated time.Time
}

type gsiForecast struct {
	Support       string `json:"support"`
	License       string `json:"license"`
	Info          string `json:"info"`
	Documentation string `json:"documentation"`
	Commercial    string `json:"commercial"`
	Signee        string `json:"signee"`
	Forecast      []struct {
		Epochtime     int     `json:"epochtime"`
		Eevalue       int     `json:"eevalue"`
		Ewind         int     `json:"ewind"`
		Esolar        int     `json:"esolar"`
		Ensolar       int     `json:"ensolar"`
		Enwind        int     `json:"enwind"`
		Sci           int     `json:"sci"`
		Gsi           float64 `json:"gsi"`
		TimeStamp     int64   `json:"timeStamp"`
		Energyprice   string  `json:"energyprice"`
		Co2GStandard  int     `json:"co2_g_standard"`
		Co2GOekostrom int     `json:"co2_g_oekostrom"`
		Timeframe     struct {
			Start int64 `json:"start"`
			End   int64 `json:"end"`
		} `json:"timeframe"`
		Iat       int64  `json:"iat"`
		Zip       string `json:"zip"`
		Signature string `json:"signature"`
	} `json:"forecast"`
	Location struct {
		Zip       string `json:"zip"`
		City      string `json:"city"`
		Signature string `json:"signature"`
	} `json:"location"`
	Err     bool
	Message string
}

var _ api.Tariff = (*GrünStromIndex)(nil)

func init() {
	registry.Add("grünstromindex", NewGrünStromIndexFromConfig)
}

func NewGrünStromIndexFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Zip string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("gsi").Redact(cc.Zip)

	t := &GrünStromIndex{
		log:    log,
		Helper: request.NewHelper(log),
		zip:    cc.Zip,
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *GrünStromIndex) run(done chan error) {
	var once sync.Once
	uri := fmt.Sprintf("https://api.corrently.io/v2.0/gsi/prediction?zip=%s", t.zip)

	for ; true; <-time.NewTicker(time.Hour).C {
		var res gsiForecast
		err := t.GetJSON(uri, &res)
		if err == nil && res.Err {
			err = errors.New(res.Message)
		}

		if err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		t.mux.Lock()
		t.updated = time.Now()

		t.data = make(api.Rates, 0, len(res.Forecast))
		for _, r := range res.Forecast {
			t.data = append(t.data, api.Rate{
				Price: 100 - r.Gsi, // gsi to cost
				Start: time.UnixMilli(r.Timeframe.Start),
				End:   time.UnixMilli(r.Timeframe.End),
			})
		}

		t.mux.Unlock()
	}
}

// Rates implements the api.Tariff interface
func (t *GrünStromIndex) Rates() (api.Rates, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	return t.data, outdatedError(t.updated, time.Hour)
}
