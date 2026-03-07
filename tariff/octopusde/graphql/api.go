package graphql

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
)

// BaseURI is Octopus Energy Germany's Kraken API root.
// The implementation in this file follows the published example at https://octopusenergy.de/blog/wohnen/dynamisch-sparen-per-api
const BaseURI = "https://api.oeg-kraken.energy/v1/graphql/"

// OctopusDeGraphQLClient provides an interface for communicating with Octopus Energy Germany's Kraken platform.
type OctopusDeGraphQLClient struct {
	log *util.Logger
	*graphql.Client
	accountNumber string
}

// NewClient returns a new, authenticated instance of OctopusDeGraphQLClient.
func NewClient(log *util.Logger, email, password, accountNumber string) (*OctopusDeGraphQLClient, error) {
	ts := oauth2.ReuseTokenSource(nil, &tokenSource{
		log:      log,
		email:    email,
		password: password,
	})

	cli := request.NewClient(log)
	cli.Transport = &transport.Decorator{
		Decorator: func(req *http.Request) error {
			token, err := ts.Token()
			if err != nil {
				return err
			}
			// Kraken API requires Authorization header without "Bearer" prefix
			req.Header.Set("Authorization", token.AccessToken)
			return nil
		},
		Base: cli.Transport,
	}

	gq := &OctopusDeGraphQLClient{
		log:           log,
		accountNumber: accountNumber,
		Client:        graphql.NewClient(BaseURI, cli),
	}

	return gq, nil
}

// UnitRateForecast queries the current and forecast pricing for the active agreement.
// It supports three tariff types:
//   - Dynamic (Octopus Dynamic): prices from unitRateForecast
//   - Time of Use (e.g. Octopus Go): repeated daily time-slot rates generated for 7 days ahead
//   - Simple (fixed rate): a single rate covering the agreement period
func (c *OctopusDeGraphQLClient) UnitRateForecast() ([]RatePeriod, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var q getDayAheadPrices
	if err := c.Client.Query(ctx, &q, map[string]any{
		"accountNumber": c.accountNumber,
	}); err != nil {
		return nil, err
	}

	// Extract rates from the query result
	if len(q.Account.Properties) == 0 {
		return nil, errors.New("no properties found")
	}

	// Find the active agreement across all properties
	for _, property := range q.Account.Properties {
		for _, malo := range property.ElectricityMalos {
			for _, agreement := range malo.Agreements {
				if !agreement.IsActive {
					continue
				}

				// Dynamic tariff: has unitRateForecast entries with per-slot prices
				if len(agreement.UnitRateForecast) > 0 {
					rates, err := extractForecastRates(agreement.UnitRateForecast)
					if err != nil {
						return nil, err
					}
					if len(rates) > 0 {
						return rates, nil
					}
				}

				// Simple tariff: single fixed rate covering the agreement period
				if grossStr := agreement.UnitRateInformation.SimpleProductUnitRateInformation.LatestGrossUnitRateCentsPerKwh; grossStr != "" {
					return extractSimpleRate(agreement.UnitRateInformation.SimpleProductUnitRateInformation, agreement.ValidFrom, agreement.ValidTo)
				}

				// Time of Use tariff: multiple time-slot rates that repeat daily
				if touRates := agreement.UnitRateInformation.TimeOfUseProductUnitRateInformation.Rates; len(touRates) > 0 {
					c.log.TRACE.Printf("detected time-of-use tariff with %d rate slots", len(touRates))
					return generateTouRates(touRates, agreement.ValidTo)
				}
			}
		}
	}

	return nil, errors.New("no active agreement found")
}

// extractForecastRates converts dynamic-tariff unitRateForecast entries into RatePeriod values.
func extractForecastRates(forecasts []unitRateForecast) ([]RatePeriod, error) {
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
					ValidFrom:                      forecast.ValidFrom,
					ValidTo:                        forecast.ValidTo,
					LatestGrossUnitRateCentsPerKwh: grossRate,
					NetUnitRateCentsPerKwh:         netRate,
				})
			}
			continue
		}

		// Forecast that uses SimpleProductUnitRateInformation
		if grossStr := info.SimpleProductUnitRateInformation.LatestGrossUnitRateCentsPerKwh; grossStr != "" {
			netRate, err := parseFloat(info.SimpleProductUnitRateInformation.NetUnitRateCentsPerKwh)
			if err != nil {
				return nil, fmt.Errorf("failed to parse net unit rate: %w", err)
			}
			grossRate, err := parseFloat(grossStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse gross unit rate: %w", err)
			}
			rates = append(rates, RatePeriod{
				ValidFrom:                      forecast.ValidFrom,
				ValidTo:                        forecast.ValidTo,
				LatestGrossUnitRateCentsPerKwh: grossRate,
				NetUnitRateCentsPerKwh:         netRate,
			})
		}
	}
	return rates, nil
}

// extractSimpleRate converts a SimpleProductUnitRateInformation into a single RatePeriod
// covering from validFrom to validTo (or 1 year ahead when validTo is zero).
func extractSimpleRate(info simpleProductUnitRateInformation, validFrom, validTo time.Time) ([]RatePeriod, error) {
	netRate, err := parseFloat(info.NetUnitRateCentsPerKwh)
	if err != nil {
		return nil, fmt.Errorf("failed to parse net unit rate: %w", err)
	}
	grossRate, err := parseFloat(info.LatestGrossUnitRateCentsPerKwh)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gross unit rate: %w", err)
	}
	if validFrom.IsZero() {
		validFrom = time.Now()
	}
	return []RatePeriod{{
		ValidFrom:                      validFrom,
		ValidTo:                        validTo, // zero means indefinite; octopusde.go handles zero ValidTo
		LatestGrossUnitRateCentsPerKwh: grossRate,
		NetUnitRateCentsPerKwh:         netRate,
	}}, nil
}

// generateTouRates produces rate periods for a Time of Use tariff over the next 7 days
// by repeating each timeslot's activation window for each day in the planning horizon.
func generateTouRates(rates []touRate, agreementValidTo time.Time) ([]RatePeriod, error) {
	const planDays = 7
	now := time.Now()
	horizon := now.AddDate(0, 0, planDays)
	if !agreementValidTo.IsZero() && agreementValidTo.Before(horizon) {
		horizon = agreementValidTo
	}

	// Midnight of today in local time
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var result []RatePeriod
	for day := startDay; day.Before(horizon); day = day.Add(24 * time.Hour) {
		for _, r := range rates {
			grossRate, err := parseFloat(r.LatestGrossUnitRateCentsPerKwh)
			if err != nil {
				return nil, fmt.Errorf("failed to parse gross unit rate for slot %q: %w", r.TimeslotName, err)
			}
			netRate, err := parseFloat(r.NetUnitRateCentsPerKwh)
			if err != nil {
				return nil, fmt.Errorf("failed to parse net unit rate for slot %q: %w", r.TimeslotName, err)
			}

			for _, rule := range r.TimeslotActivationRules {
				fromOffset, err := parseTimeOfDay(rule.ActiveFromTime)
				if err != nil {
					return nil, fmt.Errorf("failed to parse activeFromTime %q: %w", rule.ActiveFromTime, err)
				}
				toOffset, err := parseTimeOfDay(rule.ActiveToTime)
				if err != nil {
					return nil, fmt.Errorf("failed to parse activeToTime %q: %w", rule.ActiveToTime, err)
				}

				periodStart := day.Add(fromOffset)
				var periodEnd time.Time
				switch {
				case toOffset == 0:
					// "00:00:00" as end means end of day (midnight)
					periodEnd = day.Add(24 * time.Hour)
				case toOffset < fromOffset:
					// wraps past midnight
					periodEnd = day.Add(toOffset).Add(24 * time.Hour)
				default:
					periodEnd = day.Add(toOffset)
				}

				// Skip periods entirely in the past or beyond the horizon
				if periodEnd.Before(now) || periodStart.After(horizon) {
					continue
				}

				result = append(result, RatePeriod{
					ValidFrom:                      periodStart,
					ValidTo:                        periodEnd,
					LatestGrossUnitRateCentsPerKwh: grossRate,
					NetUnitRateCentsPerKwh:         netRate,
				})
			}
		}
	}

	if len(result) == 0 {
		return nil, errors.New("time-of-use tariff has no activation rules")
	}
	return result, nil
}

// parseTimeOfDay parses a time string in "HH:MM:SS" or "HH:MM" format and returns
// the duration offset from midnight.
func parseTimeOfDay(s string) (time.Duration, error) {
	var h, m, sec int
	switch len(s) {
	case 8: // HH:MM:SS
		if _, err := fmt.Sscanf(s, "%d:%d:%d", &h, &m, &sec); err != nil {
			return 0, fmt.Errorf("invalid time format %q: %w", s, err)
		}
	case 5: // HH:MM
		if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
			return 0, fmt.Errorf("invalid time format %q: %w", s, err)
		}
	default:
		return 0, fmt.Errorf("unsupported time format %q", s)
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second, nil
}

// parseFloat parses a string to float64, handling the specific format used by Octopus API
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

