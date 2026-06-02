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
	"github.com/jinzhu/now"
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

	for tick := time.Tick(time.Hour); ; <-tick {
		var rates []RatePeriod

		if err := backoff.Retry(func() error {
			agr, err := t.gqlClient.ActiveAgreement()
			if err != nil {
				if errors.Is(err, edfGbGql.ErrAuthFailed) {
					return backoff.Permanent(err)
				}
				return backoffPermanentError(err)
			}
			rates, err = edfGbRatesForAgreement(agr, time.Now())
			return backoffPermanentError(err)
		}, bo()); err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Printf("failed to fetch unit rate forecast: %v", err)
			continue
		}

		data := make(api.Rates, 0, len(rates))
		for _, r := range rates {
			data = append(data, api.Rate{
				Start: r.ValidFrom,
				End:   r.ValidTo,
				// Rates are in pence/kWh; divide by 100 to get £/kWh
				Value: r.GrossUnitRateCentsPerKwh / 100,
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

// edfGbRatesForAgreement determines the tariff type of agr and returns the rate periods.
func edfGbRatesForAgreement(agr edfGbGql.Agreement, now time.Time) ([]RatePeriod, error) {
	horizon, err := edfGbComputeHorizon(now, agr, planDays)
	if err != nil {
		return nil, err
	}

	if len(agr.UnitRateForecast) > 0 {
		rates, err := edfGbExtractForecastRates(agr.UnitRateForecast, horizon)
		if err != nil {
			return nil, err
		}
		if len(rates) > 0 {
			return rates, nil
		}
	}

	if agr.UnitRateInformation.SimpleProductUnitRateInformation.LatestGrossUnitRateCentsPerKwh != "" {
		return edfGbSimpleRates(agr.UnitRateInformation.SimpleProductUnitRateInformation, horizon)
	}

	if touRateSlots := agr.UnitRateInformation.TimeOfUseProductUnitRateInformation.Rates; len(touRateSlots) > 0 {
		return edfGbGenerateTouRates(touRateSlots, horizon)
	}

	return nil, errors.New("unsupported tariff type for active agreement")
}

func edfGbExtractForecastRates(forecasts []edfGbGql.UnitRateForecast, horizon planningHorizon) ([]RatePeriod, error) {
	var rates []RatePeriod
	for _, forecast := range forecasts {
		info := forecast.UnitRateInformation
		if info.TimeOfUseProductUnitRateInformation.Rates != nil {
			for _, r := range info.TimeOfUseProductUnitRateInformation.Rates {
				netRate, err := strconv.ParseFloat(r.NetUnitRateCentsPerKwh, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse net unit rate: %w", err)
				}
				grossRate, err := strconv.ParseFloat(r.LatestGrossUnitRateCentsPerKwh, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse gross unit rate: %w", err)
				}
				rates = append(rates, RatePeriod{
					ValidFrom:                forecast.ValidFrom,
					ValidTo:                  forecast.ValidTo,
					GrossUnitRateCentsPerKwh: grossRate,
					NetUnitRateCentsPerKwh:   netRate,
				})
			}
			continue
		}
		if info.SimpleProductUnitRateInformation.LatestGrossUnitRateCentsPerKwh != "" {
			r, err := edfGbSimpleRates(info.SimpleProductUnitRateInformation, horizon)
			if err != nil {
				return nil, err
			}
			rates = append(rates, r...)
		}
	}
	return rates, nil
}

func edfGbSimpleRates(info edfGbGql.SimpleProductUnitRateInformation, horizon planningHorizon) ([]RatePeriod, error) {
	netRate, err := strconv.ParseFloat(info.NetUnitRateCentsPerKwh, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse net unit rate: %w", err)
	}
	grossRate, err := strconv.ParseFloat(info.LatestGrossUnitRateCentsPerKwh, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gross unit rate: %w", err)
	}
	return []RatePeriod{{
		ValidFrom:                horizon.start,
		ValidTo:                  horizon.end,
		GrossUnitRateCentsPerKwh: grossRate,
		NetUnitRateCentsPerKwh:   netRate,
	}}, nil
}

func edfGbComputeHorizon(refNow time.Time, agreement edfGbGql.Agreement, days int) (planningHorizon, error) {
	start := refNow
	end := refNow.AddDate(0, 0, days)

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

func edfGbGenerateTouRates(rates []edfGbGql.TouRate, horizon planningHorizon) ([]RatePeriod, error) {
	startDay := now.With(horizon.start).BeginningOfDay()

	var result []RatePeriod
	for day := startDay; day.Before(horizon.end); day = day.Add(24 * time.Hour) {
		for _, r := range rates {
			periods, err := edfGbRatePeriodsForDay(day, horizon, r)
			if err != nil {
				return nil, err
			}
			result = append(result, periods...)
		}
	}

	if len(result) == 0 {
		if horizon.end.Before(horizon.start) {
			return nil, errors.New("time-of-use agreement has expired")
		}
		return nil, errors.New("time-of-use tariff has no upcoming periods")
	}
	return result, nil
}

func edfGbRatePeriodsForDay(day time.Time, horizon planningHorizon, r edfGbGql.TouRate) ([]RatePeriod, error) {
	grossRate, err := strconv.ParseFloat(r.LatestGrossUnitRateCentsPerKwh, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gross unit rate for slot %q: %w", r.TimeslotName, err)
	}
	netRate, err := strconv.ParseFloat(r.NetUnitRateCentsPerKwh, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse net unit rate for slot %q: %w", r.TimeslotName, err)
	}

	var periods []RatePeriod
	for _, rule := range r.TimeslotActivationRules {
		fromOffset, err := parseTimeOfDay(rule.ActiveFromTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse activeFromTime %q: %w", rule.ActiveFromTime, err)
		}
		toOffset, err := parseTimeOfDay(rule.ActiveToTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse activeToTime %q: %w", rule.ActiveToTime, err)
		}

		start, end := computePeriod(day, fromOffset, toOffset)
		if end.Before(horizon.start) || start.After(horizon.end) {
			continue
		}

		periods = append(periods, RatePeriod{
			ValidFrom:                start,
			ValidTo:                  end,
			GrossUnitRateCentsPerKwh: grossRate,
			NetUnitRateCentsPerKwh:   netRate,
		})
	}
	return periods, nil
}
