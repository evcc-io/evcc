package tariff

import (
	"testing"
	"time"

	octoDeGql "github.com/evcc-io/evcc/tariff/octopusde/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOctopusDeConfigParse(t *testing.T) {
	validConfig := map[string]any{
		"email":         "test@example.com",
		"password":      "testpassword",
		"accountNumber": "A-12345678",
	}

	tariff, err := buildOctopusDeFromConfig(validConfig)
	require.NoError(t, err)
	require.NotNil(t, tariff)

	missingEmailConfig := map[string]any{
		"password":      "testpassword",
		"accountNumber": "A-12345678",
	}
	_, err = buildOctopusDeFromConfig(missingEmailConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing email")

	missingPasswordConfig := map[string]any{
		"email":         "test@example.com",
		"accountNumber": "A-12345678",
	}
	_, err = buildOctopusDeFromConfig(missingPasswordConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing password")

	missingAccountNumberConfig := map[string]any{
		"email":    "test@example.com",
		"password": "testpassword",
	}
	_, err = buildOctopusDeFromConfig(missingAccountNumberConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing account number")
}

// t0 is a fixed reference time (Monday midnight UTC) used across
// forecast tests so that time-of-use period generation is fully deterministic.
var t0 = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

// dynamicAgreement builds an agreement that uses a dynamic tariff:
// the unitRateForecast field contains two half-hour forecast slots with
// per-slot prices stored in TimeOfUseProductUnitRateInformation.
func dynamicAgreement() octoDeGql.Agreement {
	t1 := t0
	t2 := t0.Add(15 * time.Minute)
	t3 := t2.Add(15 * time.Minute)
	return octoDeGql.Agreement{
		IsActive: true,
		UnitRateForecast: []octoDeGql.UnitRateForecast{
			{
				ValidFrom: t1,
				ValidTo:   t2,
				UnitRateInformation: octoDeGql.ForecastUnitRateInformation{
					TimeOfUseProductUnitRateInformation: octoDeGql.TimeOfUseProductUnitRateInformation{
						Rates: []octoDeGql.Rate{
							{NetUnitRateCentsPerKwh: "10.50", LatestGrossUnitRateCentsPerKwh: "12.495"},
						},
					},
				},
			},
			{
				ValidFrom: t2,
				ValidTo:   t3,
				UnitRateInformation: octoDeGql.ForecastUnitRateInformation{
					TimeOfUseProductUnitRateInformation: octoDeGql.TimeOfUseProductUnitRateInformation{
						Rates: []octoDeGql.Rate{
							{NetUnitRateCentsPerKwh: "8.00", LatestGrossUnitRateCentsPerKwh: "9.52"},
						},
					},
				},
			},
		},
	}
}

// simpleAgreement builds an agreement with a single fixed rate covering one year.
func simpleAgreement() octoDeGql.Agreement {
	return octoDeGql.Agreement{
		IsActive:  true,
		ValidFrom: t0,
		ValidTo:   t0.AddDate(1, 0, 0),
		UnitRateInformation: octoDeGql.AgreementUnitRateInformation{
			SimpleProductUnitRateInformation: octoDeGql.SimpleProductUnitRateInformation{
				NetUnitRateCentsPerKwh:         "25.00",
				LatestGrossUnitRateCentsPerKwh: "29.75",
			},
		},
	}
}

// touAgreement builds an agreement with a two-slot time-of-use tariff:
//   - Day rate  06:00–22:00
//   - Night rate 22:00–06:00 (wraps past midnight)
func touAgreement() octoDeGql.Agreement {
	return octoDeGql.Agreement{
		IsActive: true,
		UnitRateInformation: octoDeGql.AgreementUnitRateInformation{
			TimeOfUseProductUnitRateInformation: octoDeGql.TouAgreementUnitRateInformation{
				Rates: []octoDeGql.TouRate{
					{
						TimeslotName:                   "Day",
						NetUnitRateCentsPerKwh:         "30.00",
						LatestGrossUnitRateCentsPerKwh: "35.70",
						TimeslotActivationRules: []octoDeGql.TimeslotActivationRule{
							{ActiveFromTime: "06:00:00", ActiveToTime: "22:00:00"},
						},
					},
					{
						TimeslotName:                   "Night",
						NetUnitRateCentsPerKwh:         "15.00",
						LatestGrossUnitRateCentsPerKwh: "17.85",
						TimeslotActivationRules: []octoDeGql.TimeslotActivationRule{
							// 22:00 → 06:00 wraps past midnight
							{ActiveFromTime: "22:00:00", ActiveToTime: "06:00:00"},
						},
					},
				},
			},
		},
	}
}

// TestRatesForAgreement_Dynamic verifies that a dynamic tariff agreement returns
// one RatePeriod per forecast entry, preserving ValidFrom/ValidTo and rates.
func TestRatesForAgreement_Dynamic(t *testing.T) {
	rates, err := ratesForAgreement(dynamicAgreement(), t0)
	require.NoError(t, err)
	require.Len(t, rates, 2)

	assert.Equal(t, t0, rates[0].ValidFrom)
	assert.Equal(t, t0.Add(15*time.Minute), rates[0].ValidTo)
	assert.InDelta(t, 10.50, rates[0].NetUnitRateCentsPerKwh, 0.001)
	assert.InDelta(t, 12.495, rates[0].GrossUnitRateCentsPerKwh, 0.001)

	assert.Equal(t, t0.Add(15*time.Minute), rates[1].ValidFrom)
	assert.Equal(t, t0.Add(30*time.Minute), rates[1].ValidTo)
	assert.InDelta(t, 8.00, rates[1].NetUnitRateCentsPerKwh, 0.001)
	assert.InDelta(t, 9.52, rates[1].GrossUnitRateCentsPerKwh, 0.001)
}

// TestRatesForAgreement_Simple verifies that a simple fixed-rate agreement returns
// a single RatePeriod spanning the full agreement validity window.
func TestRatesForAgreement_Simple(t *testing.T) {
	rates, err := ratesForAgreement(simpleAgreement(), t0)
	require.NoError(t, err)
	require.Len(t, rates, 1)

	assert.Equal(t, t0, rates[0].ValidFrom)
	assert.Equal(t, t0.AddDate(1, 0, 0), rates[0].ValidTo)
	assert.InDelta(t, 25.00, rates[0].NetUnitRateCentsPerKwh, 0.001)
	assert.InDelta(t, 29.75, rates[0].GrossUnitRateCentsPerKwh, 0.001)
}

// TestRatesForAgreement_TimeOfUse verifies that a two-slot ToU tariff is expanded
// into 14 RatePeriods (7 days × 2 slots). With testNow at midnight the entire
// first day is in the future, so no period is filtered out.
//
// Expected layout per day (repeated 7 times):
//
//	rates[2n+0]: Day   [day+06:00, day+22:00]   net=30  gross=35.70
//	rates[2n+1]: Night [day+22:00, day+30:00]   net=15  gross=17.85  (wraps to next-day 06:00)
func TestRatesForAgreement_TimeOfUse(t *testing.T) {
	rates, err := ratesForAgreement(touAgreement(), t0)
	require.NoError(t, err)
	require.Len(t, rates, 14)

	// --- Day-0 day slot ---
	assert.Equal(t, t0.Add(6*time.Hour), rates[0].ValidFrom)
	assert.Equal(t, t0.Add(22*time.Hour), rates[0].ValidTo)
	assert.InDelta(t, 30.00, rates[0].NetUnitRateCentsPerKwh, 0.001)
	assert.InDelta(t, 35.70, rates[0].GrossUnitRateCentsPerKwh, 0.001)

	// --- Day-0 night slot (wraps: 22:00 → next-day 06:00 = +30h) ---
	assert.Equal(t, t0.Add(22*time.Hour), rates[1].ValidFrom)
	assert.Equal(t, t0.Add(30*time.Hour), rates[1].ValidTo)
	assert.InDelta(t, 15.00, rates[1].NetUnitRateCentsPerKwh, 0.001)
	assert.InDelta(t, 17.85, rates[1].GrossUnitRateCentsPerKwh, 0.001)

	// --- Day-6 day slot (last day within 7-day horizon) ---
	day6 := t0.Add(6 * 24 * time.Hour)
	assert.Equal(t, day6.Add(6*time.Hour), rates[12].ValidFrom)
	assert.Equal(t, day6.Add(22*time.Hour), rates[12].ValidTo)

	// --- Day-6 night slot (starts before the 7-day horizon, so included) ---
	assert.Equal(t, day6.Add(22*time.Hour), rates[13].ValidFrom)
	assert.Equal(t, day6.Add(30*time.Hour), rates[13].ValidTo)
}
