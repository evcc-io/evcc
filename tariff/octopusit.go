package tariff

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	krakengql "github.com/evcc-io/evcc/tariff/octopuskraken/graphql"
	"github.com/evcc-io/evcc/util"
)

// OctopusIt is an api.Tariff implementation for Octopus Energy Italy, reusing
// Germany's Kraken auth/transport but its own rate query (schemas diverge).
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
			agr, err := t.gqlClient.ItActiveAgreement()
			if err != nil {
				if errors.Is(err, krakengql.ErrAuthFailed) {
					return backoff.Permanent(err)
				}
				return backoffPermanentError(err)
			}
			rates, err = ratesForItAgreement(agr, time.Now())
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

// ratesForItAgreement returns a flat rate covering the planning horizon.
// Only FIXED_SINGLE_RATE products are supported - F1/F2/F3 is not implemented.
func ratesForItAgreement(agr krakengql.ItAgreement, now time.Time) ([]RatePeriod, error) {
	horizon, err := computeItHorizon(now, agr, planDays)
	if err != nil {
		return nil, err
	}

	p := agr.Product.Prices
	if p.ConsumptionChargeF2 != "" || p.ConsumptionChargeF3 != "" {
		return nil, fmt.Errorf("unsupported time-of-use product %q: F2/F3 rates are not implemented", agr.Product.Code)
	}

	rate, err := parseFloat(p.ConsumptionCharge)
	if err != nil {
		return nil, fmt.Errorf("failed to parse consumption charge: %w", err)
	}

	// prices are € per kWh; RatePeriod stores cents per kWh like the DE tariff.
	return []RatePeriod{{
		ValidFrom:                horizon.start,
		ValidTo:                  horizon.end,
		GrossUnitRateCentsPerKwh: rate * 100,
	}}, nil
}

// computeItHorizon returns the planning window, capped by the agreement's validity.
func computeItHorizon(now time.Time, agreement krakengql.ItAgreement, planDays int) (planningHorizon, error) {
	start := now
	end := now.AddDate(0, 0, planDays)

	if agreement.ValidFrom.After(end) || (!agreement.ValidTo.IsZero() && agreement.ValidTo.Before(start)) {
		return planningHorizon{}, errors.New("agreement is not valid for the planning horizon")
	}

	if agreement.ValidFrom.After(start) {
		start = agreement.ValidFrom
	}

	if !agreement.ValidTo.IsZero() && agreement.ValidTo.Before(end) {
		end = agreement.ValidTo
	}

	return planningHorizon{start: start, end: end}, nil
}
