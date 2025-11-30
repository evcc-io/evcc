package rest

import (
	"fmt"
	"strings"
	"time"
)

// ProductURI defines the location of the tariff information page. Substitute %s with tariff name.
const ProductURI = "https://api.octopus.energy/v1/products/%s/"

// RatesURI defines the location of the full tariff rates page, including speculation.
// Substitute first %s with product code, second with tariff code.
const RatesURI = ProductURI + "electricity-tariffs/%s/standard-unit-rates/"

// ConstructRatesAPIFromProductAndRegionCode returns a validly formatted, fully qualified URI to the unit rate information
// derived from the given product code and region.
func ConstructRatesAPIFromProductAndRegionCode(product string, region string) string {
	tCode := strings.ToUpper(fmt.Sprintf("E-1R-%s-%s", product, region))
	return fmt.Sprintf(RatesURI, product, tCode)
}

// ConstructRatesAPIFromTariffCode returns a validly formatted, fully qualified URI to the unit rate information
// derived from the given Tariff Code.
func ConstructRatesAPIFromTariffCode(tariff string) string {
	// Hacky bullshit, saves handling both the product and tariff codes in GQL mode.
	// Hopefully Octopus don't change how this works otherwise we might have to do this properly :(
	if len(tariff) < 7 {
		// OOB check
		return ""
	}
	pCode := tariff[5 : len(tariff)-2]
	return fmt.Sprintf(RatesURI, pCode, tariff)
}

type UnitRates struct {
	Count    uint64 `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []Rate `json:"results"`
}

// RatePaymentMethodDirectDebit is set when the rate only applies when the customer is paying with Direct Debit.
const RatePaymentMethodDirectDebit = "DIRECT_DEBIT"

// RatePaymentMethodNotDirectDebit is set when the rate only applies when the customer is paying with
// any payment means that ISN'T Direct Debit (say, pre-payment meters)
const RatePaymentMethodNotDirectDebit = "NON_DIRECT_DEBIT"

type Rate struct {
	ValidityStart     time.Time `json:"valid_from"`
	ValidityEnd       time.Time `json:"valid_to"`
	PriceInclusiveTax float64   `json:"value_inc_vat"`
	PriceExclusiveTax float64   `json:"value_exc_vat"`
	PaymentMethod     string    `json:"payment_method"`
}
