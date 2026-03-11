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
	octoDeGql "github.com/evcc-io/evcc/tariff/octopusde/graphql"
	"github.com/evcc-io/evcc/util"
)

type OctopusDe struct {
	log       *util.Logger
	gqlClient *octoDeGql.OctopusDeGraphQLClient
	data      *util.Monitor[api.Rates]
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

	log := util.NewLogger("octopus-de")

	// Create GraphQL client
	gqlClient, err := octoDeGql.NewClient(log, cc.Email, cc.Password, cc.AccountNumber)
	if err != nil {
		return nil, err
	}

	t := &OctopusDe{
		log:       log,
		gqlClient: gqlClient,
		data:      util.NewMonitor[api.Rates](2 * time.Hour),
	}

	return t, nil
}

func (t *OctopusDe) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		var rates []RatePeriod

		if err := backoff.Retry(func() error {
			agr, err := t.gqlClient.ActiveAgreement()
			if err != nil {
				return backoffPermanentError(err)
			}
			rates, err = ratesForAgreement(agr, time.Now())
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
				Value: r.GrossUnitRateCentsPerKwh / 100,
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

// RatePeriod represents a parsed rate period with pricing in cents per kWh.
type RatePeriod struct {
	ValidFrom                time.Time
	ValidTo                  time.Time
	NetUnitRateCentsPerKwh   float64
	GrossUnitRateCentsPerKwh float64
}

// ratesForAgreement determines the tariff type of agr and returns the corresponding
// rate periods. It supports Dynamic, Simple, and Time-of-Use tariffs.
// now is used as the reference time for ToU rate generation.
func ratesForAgreement(agr octoDeGql.Agreement, now time.Time) ([]RatePeriod, error) {
	// Dynamic tariff: has unitRateForecast entries with per-slot prices
	if len(agr.UnitRateForecast) > 0 {
		rates, err := extractForecastRates(agr.UnitRateForecast)
		if err != nil {
			return nil, err
		}
		if len(rates) > 0 {
			return rates, nil
		}
	}

	// Simple tariff: single fixed rate covering the agreement period
	if agr.UnitRateInformation.SimpleProductUnitRateInformation.LatestGrossUnitRateCentsPerKwh != "" {
		return simpleRates(agr.UnitRateInformation.SimpleProductUnitRateInformation, agr.ValidFrom, agr.ValidTo)
	}

	// Time of Use tariff: multiple time-slot rates that repeat daily
	if touRateSlots := agr.UnitRateInformation.TimeOfUseProductUnitRateInformation.Rates; len(touRateSlots) > 0 {
		return generateTouRates(touRateSlots, agr.ValidTo, now)
	}

	return nil, errors.New("unsupported tariff type for active agreement")
}

// extractForecastRates converts dynamic-tariff UnitRateForecast entries into RatePeriod values.
func extractForecastRates(forecasts []octoDeGql.UnitRateForecast) ([]RatePeriod, error) {
	var rates []RatePeriod
	for _, forecast := range forecasts {
		info := forecast.UnitRateInformation

		// Dynamic forecasts typically use TimeOfUseProductUnitRateInformation
		if info.TimeOfUseProductUnitRateInformation.Rates != nil {
			for _, r := range info.TimeOfUseProductUnitRateInformation.Rates {
				netRate, err := parseFloat(r.NetUnitRateCentsPerKwh)
				if err != nil {
					return nil, fmt.Errorf("failed to parse net unit rate: %w", err)
				}
				grossRate, err := parseFloat(r.LatestGrossUnitRateCentsPerKwh)
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

		// Forecast that uses SimpleProductUnitRateInformation
		if info.SimpleProductUnitRateInformation.LatestGrossUnitRateCentsPerKwh != "" {
			r, err := simpleRates(info.SimpleProductUnitRateInformation, forecast.ValidFrom, forecast.ValidTo)
			if err != nil {
				return nil, err
			}
			rates = append(rates, r...)
		}
	}
	return rates, nil
}

// simpleRates converts a SimpleProductUnitRateInformation into a single RatePeriod
// covering from to to. A zero to means indefinite; run() handles zero ValidTo.
func simpleRates(info octoDeGql.SimpleProductUnitRateInformation, from, to time.Time) ([]RatePeriod, error) {
	netRate, err := parseFloat(info.NetUnitRateCentsPerKwh)
	if err != nil {
		return nil, fmt.Errorf("failed to parse net unit rate: %w", err)
	}
	grossRate, err := parseFloat(info.LatestGrossUnitRateCentsPerKwh)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gross unit rate: %w", err)
	}
	if from.IsZero() {
		from = time.Now()
	}
	return []RatePeriod{{
		ValidFrom:                from,
		ValidTo:                  to, // zero means indefinite; run() handles zero ValidTo
		GrossUnitRateCentsPerKwh: grossRate,
		NetUnitRateCentsPerKwh:   netRate,
	}}, nil
}

// computeHorizon returns the end of the planning window, capped by agreementValidTo.
func computeHorizon(now, agreementValidTo time.Time, planDays int) time.Time {
	h := now.AddDate(0, 0, planDays)
	if !agreementValidTo.IsZero() && agreementValidTo.Before(h) {
		return agreementValidTo
	}
	return h
}

// computePeriod converts day-relative time offsets into absolute start/end times,
// handling the midnight-wrapping convention ("00:00:00" means end-of-day).
func computePeriod(day time.Time, fromOffset, toOffset time.Duration) (time.Time, time.Time) {
	start := day.Add(fromOffset)
	var end time.Time
	switch {
	case toOffset == 0:
		// "00:00:00" as end means end of day (midnight)
		end = day.Add(24 * time.Hour)
	case toOffset < fromOffset:
		// wraps past midnight
		end = day.Add(toOffset).Add(24 * time.Hour)
	default:
		end = day.Add(toOffset)
	}
	return start, end
}

// ratePeriodsForDay expands one TouRate slot for a single day into RatePeriods,
// filtered to the window [now, horizon].
func ratePeriodsForDay(day, now, horizon time.Time, r octoDeGql.TouRate) ([]RatePeriod, error) {
	grossRate, err := parseFloat(r.LatestGrossUnitRateCentsPerKwh)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gross unit rate for slot %q: %w", r.TimeslotName, err)
	}
	netRate, err := parseFloat(r.NetUnitRateCentsPerKwh)
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
		if end.Before(now) || start.After(horizon) {
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

// generateTouRates produces rate periods for a Time of Use tariff over the next 7 days
// by repeating each timeslot's activation window for each day in the planning horizon.
// now is the reference time used for filtering past periods and computing the horizon.
func generateTouRates(rates []octoDeGql.TouRate, agreementValidTo time.Time, now time.Time) ([]RatePeriod, error) {
	const planDays = 7
	horizon := computeHorizon(now, agreementValidTo, planDays)
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var result []RatePeriod
	for day := startDay; day.Before(horizon); day = day.Add(24 * time.Hour) {
		for _, r := range rates {
			dayPeriods, err := ratePeriodsForDay(day, now, horizon, r)
			if err != nil {
				return nil, err
			}
			result = append(result, dayPeriods...)
		}
	}

	if len(result) == 0 {
		if !agreementValidTo.IsZero() && agreementValidTo.Before(now) {
			return nil, errors.New("time-of-use agreement has expired")
		}
		return nil, errors.New("time-of-use tariff has no upcoming periods")
	}
	return result, nil
}

// parseTimeOfDay parses a time string in "HH:MM:SS" or "HH:MM" format and returns
// the duration offset from midnight.
func parseTimeOfDay(s string) (time.Duration, error) {
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return time.Duration(t.Hour())*time.Hour +
				time.Duration(t.Minute())*time.Minute +
				time.Duration(t.Second())*time.Second, nil
		}
	}
	return 0, fmt.Errorf("unsupported time format %q", s)
}

// parseFloat parses a string to float64.
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
