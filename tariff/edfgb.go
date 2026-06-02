package tariff

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	edfGbGql "github.com/evcc-io/evcc/tariff/edfgb/graphql"
	"github.com/evcc-io/evcc/util"
)

type EdfGb struct {
	log       *util.Logger
	gqlClient *edfGbGql.EdfGbGraphQLClient
	data      *util.Monitor[api.Rates]
}

var _ api.Tariff = (*EdfGb)(nil)

func init() {
	registry.Add("edf-gb", NewEdfGbFromConfig)
}

// NewEdfGbFromConfig creates the tariff provider from the given config map, and runs it.
func NewEdfGbFromConfig(other map[string]any) (api.Tariff, error) {
	t, err := buildEdfGbFromConfig(other)
	if err != nil {
		return nil, err
	}
	return runOrError(t)
}

func buildEdfGbFromConfig(other map[string]any) (*EdfGb, error) {
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

	log := util.NewLogger("edf-gb")

	gqlClient, err := edfGbGql.NewClient(log, cc.Email, cc.Password, cc.AccountNumber)
	if err != nil {
		return nil, err
	}

	return &EdfGb{
		log:       log,
		gqlClient: gqlClient,
		data:      util.NewMonitor[api.Rates](2 * time.Hour),
	}, nil
}

func (t *EdfGb) run(done chan error) {
	var once sync.Once

	// Discover MPAN once at startup.
	var mpan string
	if err := backoff.Retry(func() error {
		var err error
		mpan, err = t.gqlClient.MPAN()
		if err != nil {
			if errors.Is(err, edfGbGql.ErrAuthFailed) {
				return backoff.Permanent(err)
			}
			return backoffPermanentError(err)
		}
		return nil
	}, bo()); err != nil {
		once.Do(func() { done <- fmt.Errorf("failed to discover MPAN: %w", err) })
		return
	}

	for tick := time.Tick(time.Hour); ; <-tick {
		now := time.Now()
		startAt := now
		endAt := now.AddDate(0, 0, planDays)

		var rawRates []edfGbGql.ApplicableRate

		if err := backoff.Retry(func() error {
			var err error
			rawRates, err = t.gqlClient.Rates(mpan, startAt, endAt)
			if err != nil {
				if errors.Is(err, edfGbGql.ErrAuthFailed) {
					return backoff.Permanent(err)
				}
				return backoffPermanentError(err)
			}
			return nil
		}, bo()); err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Printf("failed to fetch unit rate forecast: %v", err)
			continue
		}

		data := make(api.Rates, 0, len(rawRates))
		for _, r := range rawRates {
			grossRate, err := strconv.ParseFloat(r.GrossUnitRateCentsPerKwh, 64)
			if err != nil {
				t.log.WARN.Printf("failed to parse rate %q: %v", r.GrossUnitRateCentsPerKwh, err)
				continue
			}
			data = append(data, api.Rate{
				Start: r.ValidFrom,
				End:   r.ValidTo,
				// Rates are in pence/kWh; divide by 100 to get £/kWh
				Value: grossRate / 100,
			})
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface.
func (t *EdfGb) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface.
func (t *EdfGb) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
