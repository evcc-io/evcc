package tariff

import (
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	octoDeGql "github.com/evcc-io/evcc/tariff/octopusde/graphql"
	"github.com/evcc-io/evcc/util"
)

type OctopusDe struct {
	log           *util.Logger
	email         string
	password      string
	accountNumber string
	data          *util.Monitor[api.Rates]
}

var _ api.Tariff = (*OctopusDe)(nil)

func init() {
	registry.Add("octopus-de", NewOctopusDeFromConfig)
}

// NewOctopusDeFromConfig creates the tariff provider from the given config map, and runs it.
func NewOctopusDeFromConfig(other map[string]any) (api.Tariff, error) {
	t, err := buildOctopusDeFromConfig(other)
	if err != nil {
		return nil, err
	}

	return runOrError(t)
}

// buildOctopusDeFromConfig creates the Tariff provider from the given config map.
// Split out to allow for testing.
func buildOctopusDeFromConfig(other map[string]any) (*OctopusDe, error) {
	var cc struct {
		Email         string
		Password      string
		AccountNumber string
	}

	logger := util.NewLogger("octopus-de")

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Email == "" {
		return nil, errors.New("missing email")
	}

	if cc.Password == "" {
		return nil, errors.New("missing password")
	}

	if cc.AccountNumber == "" {
		return nil, errors.New("missing account number")
	}

	t := &OctopusDe{
		log:           logger,
		email:         cc.Email,
		password:      cc.Password,
		accountNumber: cc.AccountNumber,
		data:          util.NewMonitor[api.Rates](2 * time.Hour),
	}

	return t, nil
}

func (t *OctopusDe) run(done chan error) {
	var once sync.Once

	// Create GraphQL client
	gqlClient, err := octoDeGql.NewClient(t.log, t.email, t.password, t.accountNumber)
	if err != nil {
		once.Do(func() { done <- err })
		t.log.ERROR.Println(err)
		return
	}

	for tick := time.Tick(time.Hour); ; <-tick {
		var rates []octoDeGql.RatePeriod

		if err := backoff.Retry(func() error {
			var err error
			rates, err = gqlClient.UnitRateForecast()
			return backoffPermanentError(err)
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Printf("failed to fetch unit rate forecast: %v", err)
			continue
		}

		data := make(api.Rates, 0, len(rates))
		for _, r := range rates {
			// ValidTo can be zero which means the rate has no expected end
			// Set it to a date far in the future in this case
			rateEnd := r.ValidTo
			if rateEnd.IsZero() {
				t.log.TRACE.Printf("handling rate with indefinite length: %v", r.ValidFrom)
				// Add a year from the start date
				rateEnd = r.ValidFrom.AddDate(1, 0, 0)
			}
			ar := api.Rate{
				Start: r.ValidFrom,
				End:   rateEnd,
				// Convert from cents per kWh to price per kWh (divide by 100)
				// Use gross price (including tax) as that's what the customer pays
				Value: r.LatestGrossUnitRateCentsPerKwh / 100,
			}
			data = append(data, ar)
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *OctopusDe) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *OctopusDe) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
