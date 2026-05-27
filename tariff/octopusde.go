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
	"github.com/jinzhu/now"
)

// ErrAuthFailed re-exports the GraphQL auth-failure sentinel for use in tests.
var ErrAuthFailed = octoDeGql.ErrAuthFailed

type OctopusDe struct {
	log       *util.Logger
	gqlClient *octoDeGql.OctopusDeGraphQLClient
	data      *util.Monitor[api.Rates]
}

type planningHorizon struct {
	start time.Time
	end   time.Time
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
				if errors.Is(err, octoDeGql.ErrAuthFailed) {
					return backoff.Permanent(err)
				}
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

// planDays is the planning horizon used for all tariff types.
const planDays = 7

// RatePeriod represents a parsed rate period with pricing in cents per kWh.
type RatePeriod struct {
	ValidFrom                time.Time
	ValidTo                  time.Time
	NetUnitRateCentsPerKwh   float64
	GrossUnitRateCentsPerKwh float64
}

// ratesForAgreement determines the tariff type of agr and returns the corresponding
// rate periods. It supports Dynamic, Simple, and Time-of-Use tariffs.
// now is used as the reference time for horizon computation and ToU rate generation.
func ratesForAgreement(agr octoDeGql.Agreement, now time.Time) ([]RatePeriod, error) {
	horizon, err := computeHorizon(now, agr, planDays)
	if err != nil {
		return nil, err
	}

	// Dynamic tariff: has unitRateForecast entries with per-slot prices
	if len(agr.UnitRateForecast) > 0 {
		rates, err := extractForecastRates(agr.UnitRateForecast, horizon)
		if err != nil {
			return nil, err
		}
		if len(rates) > 0 {
			return rates, nil
		}
	}

	// Simple tariff: single fixed rate covering the agreement period
	if agr.UnitRateInformation.SimpleProductUnitRateInformation.LatestGrossUnitRateCentsPerKwh != "" {
		return simpleRates(agr.UnitRateInformation.SimpleProductUnitRateInformation, horizon)
	}

	// Time of Use tariff: multiple time-slot rates that repeat daily
	if touRateSlots := agr.UnitRateInformation.TimeOfUseProductUnitRateInformation.Rates; len(touRateSlots) > 0 {
		return generateTouRates(touRateSlots, horizon)
	}

	return nil, errors.New("unsupported tariff type for active agreement")
}

// extractForecastRates converts dynamic-tariff UnitRateForecast entries into RatePeriod values.
func extractForecastRates(forecasts []octoDeGql.UnitRateForecast, horizon planningHorizon) ([]RatePeriod, error) {
	var rates []RatePeriod
	for _, forecast := range forecasts {
		info := forecast.UnitRateInformation

		// Dynamic forecasts typically use TimeOfUseProductUnitRateInformation
		// We do expect that octopus will always return us data that falls within the planning horizon here
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
			r, err := simpleRates(info.SimpleProductUnitRateInformation, horizon)
			if err != nil {
				return nil, err
			}
			rates = append(rates, r...)
		}
	}
	return rates, nil
}

// simpleRates converts a SimpleProductUnitRateInformation into a single RatePeriod
// ending at horizon, the pre-computed planning horizon.
func simpleRates(info octoDeGql.SimpleProductUnitRateInformation, horizon planningHorizon) ([]RatePeriod, error) {
	netRate, err := parseFloat(info.NetUnitRateCentsPerKwh)
	if err != nil {
		return nil, fmt.Errorf("failed to parse net unit rate: %w", err)
	}
	grossRate, err := parseFloat(info.LatestGrossUnitRateCentsPerKwh)
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

// computeHorizon returns the planning window, capped by the validity of the agreement.
func computeHorizon(now time.Time, agreement octoDeGql.Agreement, planDays int) (planningHorizon, error) {
	start := now
	end := now.AddDate(0, 0, planDays)

	// Validate agreement overlaps with planning horizon
	if agreement.ValidFrom.After(end) || (!agreement.ValidTo.IsZero() && agreement.ValidTo.Before(start)) {
		return planningHorizon{}, errors.New("agreement is not valid for the planning horizon")
	}

	// Cap the horizon to agreement validity period
	if agreement.ValidFrom.After(start) {
		start = agreement.ValidFrom
	}

	// validTo may be unset if the agreement has no defined end yet (ie. automatically renewed)
	if !agreement.ValidTo.IsZero() && agreement.ValidTo.Before(end) {
		end = agreement.ValidTo
	}

	return planningHorizon{start: start, end: end}, nil
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
func ratePeriodsForDay(day time.Time, horizon planningHorizon, r octoDeGql.TouRate) ([]RatePeriod, error) {
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

// generateTouRates produces rate periods for a Time of Use tariff
// by repeating each timeslot's activation window for each day in the planning horizon.
// now is the reference time for filtering past periods; horizon is the pre-computed end of the window.
func generateTouRates(rates []octoDeGql.TouRate, horizon planningHorizon) ([]RatePeriod, error) {
	startDay := now.With(horizon.start).BeginningOfDay()

	var result []RatePeriod
	for day := startDay; day.Before(horizon.end); day = day.Add(24 * time.Hour) {
		for _, r := range rates {
			dayPeriods, err := ratePeriodsForDay(day, horizon, r)
			if err != nil {
				return nil, err
			}
			result = append(result, dayPeriods...)
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
