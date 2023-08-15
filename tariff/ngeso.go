package tariff

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/ngeso"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/exp/slices"
)

type Ngeso struct {
	mux            sync.Mutex
	log            *util.Logger
	regionId       string
	regionPostcode string
	data           api.Rates
	updated        time.Time
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
		return nil, errors.New("region and postcode cannot be defined simultaneously - pick one")
	}

	t := &Ngeso{
		log:            util.NewLogger("ngeso"),
		regionId:       cc.Region,
		regionPostcode: cc.Postcode,
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
	tUri := ngeso.ConstructForecastNationalAPI()
	var apiDecodeAsRegional bool

	// If a region is available, use that.
	// These should never be set simultaneously (see NewNgesoFromConfig), but in the rare case that they are,
	// use the postcode as the preferred method.
	// Regional responses are subtly different, so we set a flag to let the loop know to decode them as such. FIXME?
	if t.regionId != "" {
		tUri = ngeso.ConstructForecastRegionalByIdAPI(t.regionId)
		apiDecodeAsRegional = true
	}
	if t.regionPostcode != "" {
		tUri = ngeso.ConstructForecastRegionalByPostcodeAPI(t.regionPostcode)
		apiDecodeAsRegional = true
	}

	// Data updated by ESO every half hour, but we only need data every hour to stay current.
	for ; true; <-time.Tick(time.Hour) {
		// FIXME Eww, this is a sloppy way of handling this. Maybe we should move abstraction of this to the ngeso package?
		var rgnlWrapper ngeso.RegionalIntensityResult
		var res ngeso.IntensityRates

		if err := backoff.Retry(func() error {
			var wErr error
			// (vomits internally)
			if apiDecodeAsRegional {
				wErr = client.GetJSON(tUri, &rgnlWrapper)
				if wErr == nil {
					res = rgnlWrapper.Results
				}
			} else {
				wErr = client.GetJSON(tUri, &res)
			}
			if wErr != nil {
				// Catch cases where we're sending completely incorrect data (usually the result of a bad region).
				switch e := wErr.(type) {
				case request.StatusError:
					if e.StatusCode() == http.StatusBadRequest {
						return backoff.Permanent(wErr)
					}
				}
			}
			return wErr
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		t.mux.Lock()
		t.updated = time.Now()

		t.data = make(api.Rates, 0, len(res.Results))
		for _, r := range res.Results {
			ar := api.Rate{
				Start: r.ValidityStart.Time,
				End:   r.ValidityEnd.Time,
				// Use the forecasted rate, as the actual rate is only available for historical data
				Price: r.Intensity.Forecast,
			}
			t.data = append(t.data, ar)
		}

		t.mux.Unlock()
	}
}

// Rates implements the api.Tariff interface
func (t *Ngeso) Rates() (api.Rates, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	return slices.Clone(t.data), outdatedError(t.updated, time.Hour)
}

// Type implements the api.Tariff interface
func (t *Ngeso) Type() api.TariffType {
	return api.TariffTypeCo2
}
