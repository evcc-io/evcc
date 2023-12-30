package tariff

import (
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/ngeso"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Ngeso struct {
	log            *util.Logger
	regionId       string
	regionPostcode string
	data           *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Ngeso)(nil)

func init() {
	registry.Add("ngeso", NewNgesoFromConfig)
}

func NewNgesoFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Region   string
		Postcode string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Region != "" && cc.Postcode != "" {
		return nil, errors.New("cannot define region and postcode simultaneously")
	}

	t := &Ngeso{
		log:            util.NewLogger("ngeso"),
		regionId:       cc.Region,
		regionPostcode: cc.Postcode,
		data:           util.NewMonitor[api.Rates](2 * time.Hour),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Ngeso) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)
	bo := newBackoff()

	// Use national results by default.
	var tReq ngeso.CarbonForecastRequest
	tReq = ngeso.ConstructNationalForecastRequest()

	// If a region is available, use that.
	// These should never be set simultaneously (see NewNgesoFromConfig), but in the rare case that they are,
	// use the postcode as the preferred method.
	if t.regionId != "" {
		tReq = ngeso.ConstructRegionalForecastByIDRequest(t.regionId)
	}
	if t.regionPostcode != "" {
		tReq = ngeso.ConstructRegionalForecastByPostcodeRequest(t.regionPostcode)
	}

	// Data updated by ESO every half hour, but we only need data every hour to stay current.
	for ; true; <-time.Tick(time.Hour) {
		var carbonResponse ngeso.CarbonForecastResponse
		if err := backoff.Retry(func() error {
			var err error
			carbonResponse, err = tReq.DoRequest(client)

			// Consider whether errors.As would be more appropriate if this needs to start dealing with wrapped errors.
			if se, ok := err.(request.StatusError); ok && se.HasStatus(http.StatusBadRequest) {
				// Catch cases where we're sending completely incorrect data (usually the result of a bad region).
				return backoff.Permanent(se)
			}
			return err
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(carbonResponse.Results()))
		for _, r := range carbonResponse.Results() {
			ar := api.Rate{
				Start: r.ValidityStart.Time,
				End:   r.ValidityEnd.Time,
				// Use the forecasted rate, as the actual rate is only available for historical data
				Price: r.Intensity.Forecast,
			}
			data = append(data, ar)
		}
		data.Sort()

		t.data.Set(data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *Ngeso) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Ngeso) Type() api.TariffType {
	return api.TariffTypeCo2
}
