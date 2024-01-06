package octopus

import (
	"fmt"
	"strings"
	"time"
)

// ProductURI defines the location of the tariff information page. Substitute %s with tariff name.
const ProductURI = "https://api.octopus.energy/v1/products/%s/"

// RatesURI defines the location of the full tariff rates page, including speculation.
// Substitute first %s with tariff name, second with region code.
const RatesURI = ProductURI + "electricity-tariffs/E-1R-%s-%s/standard-unit-rates/"

// ConstructRatesAPI returns a validly formatted, fully qualified URI to the unit rate information.
func ConstructRatesAPI(tariff string, region string) string {
	t := strings.ToUpper(tariff)
	r := strings.ToUpper(region)
	return fmt.Sprintf(RatesURI, t, t, r)
}

type UnitRates struct {
	Count    uint64 `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []Rate `json:"results"`
}

type Rate struct {
	ValidityStart     time.Time `json:"valid_from"`
	ValidityEnd       time.Time `json:"valid_to"`
	PriceInclusiveTax float64   `json:"value_inc_vat"`
	PriceExclusiveTax float64   `json:"value_exc_vat"`
}
