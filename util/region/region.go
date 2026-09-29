// Package region provides helpers derived from the ISO 3166-1 alpha-2 country
// code stored as the evcc instance's site country.
package region

import (
	"fmt"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
)

// Parse validates code as an ISO 3166-1 alpha-2 country code and returns the
// corresponding language.Region.
func Parse(code string) (language.Region, error) {
	region, err := language.ParseRegion(code)
	if err != nil || !region.IsCountry() || region.String() != code {
		return language.Region{}, fmt.Errorf("invalid country code: %s", code)
	}
	return region, nil
}

// Currency returns the currency unit that is currently legal tender in the
// given country, if any. It reports false if code is not a valid country or
// has no single current legal tender currency.
func Currency(code string) (currency.Unit, bool) {
	region, err := Parse(code)
	if err != nil {
		return currency.Unit{}, false
	}
	return currency.FromRegion(region)
}
