package tariff

import (
	"testing"
	"time"

	krakengql "github.com/evcc-io/evcc/tariff/octopuskraken/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOctopusItConfigParse(t *testing.T) {
	validConfig := map[string]any{
		"email":         "test@example.com",
		"password":      "testpassword",
		"accountNumber": "A-12345678",
	}

	tariff, err := buildOctopusItFromConfig(validConfig)
	require.NoError(t, err)
	require.NotNil(t, tariff)

	missingEmailConfig := map[string]any{
		"password":      "testpassword",
		"accountNumber": "A-12345678",
	}
	_, err = buildOctopusItFromConfig(missingEmailConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing email")

	missingPasswordConfig := map[string]any{
		"email":         "test@example.com",
		"accountNumber": "A-12345678",
	}
	_, err = buildOctopusItFromConfig(missingPasswordConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing password")

	missingAccountNumberConfig := map[string]any{
		"email":    "test@example.com",
		"password": "testpassword",
	}
	_, err = buildOctopusItFromConfig(missingAccountNumberConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing account number")
}

// fixedItAgreement builds a FIXED_SINGLE_RATE agreement matching the shape
// confirmed against the live Kraken IT API (issue #31505).
func fixedItAgreement(validFrom, validTo time.Time) krakengql.ItAgreement {
	agr := krakengql.ItAgreement{
		IsActive:  true,
		ValidFrom: validFrom,
		ValidTo:   validTo,
	}
	agr.Product.Code = "000129ESFML09XXXXXXXXOCTOFIXv109"
	agr.Product.Prices = krakengql.ElectricityProductPrices{
		ProductType:       "FIXED_SINGLE_RATE",
		ConsumptionCharge: "0.09900",
	}
	return agr
}

func TestRatesForItAgreementFixed(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	validFrom := now.AddDate(0, -9, 0)
	validTo := now.AddDate(0, 3, 0)
	agr := fixedItAgreement(validFrom, validTo)

	rates, err := ratesForItAgreement(agr, now)
	require.NoError(t, err)
	require.Len(t, rates, 1)

	r := rates[0]
	assert.Equal(t, now, r.ValidFrom)
	assert.Equal(t, now.AddDate(0, 0, planDays), r.ValidTo)
	assert.InDelta(t, 9.9, r.GrossUnitRateCentsPerKwh, 1e-9)
}

func TestRatesForItAgreementCapsToAgreementValidity(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	validTo := now.AddDate(0, 0, 2)
	agr := fixedItAgreement(now.AddDate(0, -1, 0), validTo)

	rates, err := ratesForItAgreement(agr, now)
	require.NoError(t, err)
	require.Len(t, rates, 1)
	assert.Equal(t, validTo, rates[0].ValidTo)
}

func TestRatesForItAgreementRejectsTimeOfUse(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	agr := fixedItAgreement(now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))
	agr.Product.Prices.ConsumptionChargeF2 = "0.12000"
	agr.Product.Prices.ConsumptionChargeF3 = "0.07000"

	_, err := ratesForItAgreement(agr, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "time-of-use")
}
