package tariff

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

type ElectricityMaps struct {
	*request.Helper
	log     *util.Logger
	mux     sync.Mutex
	uri     string
	zone    string
	data    []CarbonIntensitySlot
	updated time.Time
}

type CarbonIntensity struct {
	Error    string
	Zone     string
	Forecast []CarbonIntensitySlot
}

type CarbonIntensitySlot struct {
	CarbonIntensity float64   // 626,
	Datetime        time.Time // "2022-12-12T16:00:00.000Z"
}

var _ api.Tariff = (*ElectricityMaps)(nil)

func init() {
	registry.Add("electricitymaps", NewElectricityMapsFromConfig)
}

func NewElectricityMapsFromConfig(other map[string]interface{}) (api.Tariff, error) {
	cc := struct {
		Uri   string
		Token string
		Zone  string
	}{
		Zone: "DE",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("em").Redact(cc.Token)

	t := &ElectricityMaps{
		log:    log,
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimRight(cc.Uri, "/"), "https"),
		zone:   strings.ToUpper(cc.Zone),
	}

	t.Client.Transport = &transport.Decorator{
		Base: t.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			"X-BLOBR-KEY": cc.Token,
		}),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *ElectricityMaps) run(done chan error) {
	var once sync.Once
	uri := fmt.Sprintf("%s/carbon-intensity/forecast?zone=%s", t.uri, t.zone)

	for ; true; <-time.Tick(time.Hour) {
		var res CarbonIntensity
		if err := t.GetJSON(uri, &res); err != nil {
			if res.Error != "" {
				err = errors.New(res.Error)
			}

			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		t.mux.Lock()
		t.updated = time.Now()
		t.data = res.Forecast
		t.mux.Unlock()
	}
}

// Unit implements the api.Tariff interface
func (t *ElectricityMaps) Unit() string {
	return Co2Equivalent
}

func (t *ElectricityMaps) Rates() (api.Rates, error) {
	t.mux.Lock()
	defer t.mux.Unlock()

	res := make(api.Rates, 0, len(t.data))
	for _, r := range t.data {
		ar := api.Rate{
			Start: r.Datetime.Local(),
			End:   r.Datetime.Add(time.Hour).Local(),
			Price: r.CarbonIntensity,
		}
		res = append(res, ar)
	}

	return res, outdatedError(t.updated, time.Hour)
}
