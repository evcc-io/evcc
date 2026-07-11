package tariff

import (
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	krakengql "github.com/evcc-io/evcc/tariff/octopuskraken/graphql"
	"github.com/evcc-io/evcc/util"
)

// OctopusIt is an api.Tariff implementation for Octopus Energy Italy, reusing
// the Germany implementation's Kraken GraphQL client and rate computation.
type OctopusIt struct {
	log       *util.Logger
	gqlClient *krakengql.Client
	data      *util.Monitor[api.Rates]
}

var _ api.Tariff = (*OctopusIt)(nil)

func init() {
	registry.Add("octopus-it", NewOctopusItFromConfig)
}

// NewOctopusItFromConfig creates the tariff provider from the given config map, and runs it.
func NewOctopusItFromConfig(other map[string]any) (api.Tariff, error) {
	t, err := buildOctopusItFromConfig(other)
	if err != nil {
		return nil, err
	}

	return runOrError(t)
}

// buildOctopusItFromConfig creates the Tariff provider from the given config map.
// Split out to allow for testing.
func buildOctopusItFromConfig(other map[string]any) (*OctopusIt, error) {
	var cc struct {
		Email         string
		Password      string
		AccountNumber string
	}

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

	log := util.NewLogger("octopus-it")

	gqlClient, err := krakengql.NewClient(log, krakengql.ItBaseURI, cc.Email, cc.Password, cc.AccountNumber)
	if err != nil {
		return nil, err
	}

	t := &OctopusIt{
		log:       log,
		gqlClient: gqlClient,
		data:      util.NewMonitor[api.Rates](2 * time.Hour),
	}

	return t, nil
}

func (t *OctopusIt) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		var rates []RatePeriod

		if err := backoff.Retry(func() error {
			agr, err := t.gqlClient.ActiveAgreement()
			if err != nil {
				if errors.Is(err, krakengql.ErrAuthFailed) {
					return backoff.Permanent(err)
				}
				return backoffPermanentError(err)
			}
			rates, err = ratesForAgreement(agr, time.Now())
			return backoffPermanentError(err)
		}, bo()); err != nil {
			if reportError(&once, done, err) {
				return
			}

			t.log.ERROR.Printf("failed to fetch unit rate forecast: %v", err)
			continue
		}

		data := make(api.Rates, 0, len(rates))
		for _, r := range rates {
			ar := api.Rate{
				Start: r.ValidFrom,
				End:   r.ValidTo,
				// Convert from cents per kWh to € per kWh (divide by 100)
				// Use gross price (including tax) as that's what the customer pays
				Value: r.GrossUnitRateCentsPerKwh / 100,
			}
			data = append(data, ar)
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *OctopusIt) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *OctopusIt) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
