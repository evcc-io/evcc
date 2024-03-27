package tariff

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type GrünStromIndex struct {
	log  *util.Logger
	zip  string
	data *util.Monitor[api.Rates]
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
	Message any
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
		log:  log,
		zip:  cc.Zip,
		data: util.NewMonitor[api.Rates](2 * time.Hour),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *GrünStromIndex) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)
	bo := newBackoff()
	uri := fmt.Sprintf("https://api.corrently.io/v2.0/gsi/prediction?zip=%s", t.zip)

	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		var res gsiForecast

		err := backoff.Retry(func() error {
			return client.GetJSON(uri, &res)
		}, bo)

		if err == nil && res.Err {
			if s, ok := res.Message.(string); ok {
				err = errors.New(s)
			} else {
				err = api.ErrNotAvailable
			}
		}

		if err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res.Forecast))
		for _, r := range res.Forecast {
			data = append(data, api.Rate{
				Price: float64(r.Co2GStandard),
				Start: time.UnixMilli(r.Timeframe.Start).Local(),
				End:   time.UnixMilli(r.Timeframe.End).Local(),
			})
		}
		data.Sort()

		t.data.Set(data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *GrünStromIndex) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *GrünStromIndex) Type() api.TariffType {
	return api.TariffTypeCo2
}
