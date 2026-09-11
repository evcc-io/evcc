package rest

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	forecastDuration = 48 * time.Hour
	rateSlotDuration = 30 * time.Minute
)

type jsonGetter interface {
	GetJSON(string, any) error
}

// Client provides access to Octopus Energy tariff rates.
type Client struct {
	getter  jsonGetter
	baseURI string
}

// NewClient creates an Octopus Energy REST client.
func NewClient(getter jsonGetter) *Client {
	return newClient(getter, "https://api.octopus.energy/v1")
}

func newClient(getter jsonGetter, baseURI string) *Client {
	return &Client{getter: getter, baseURI: strings.TrimRight(baseURI, "/")}
}

// UnitRates returns the tariff rates for the given product and tariff code.
func (c *Client) UnitRates(productCode, tariffCode string, now time.Time) (UnitRates, error) {
	var res UnitRates
	if err := c.getter.GetJSON(fmt.Sprintf("%s/products/%s/electricity-tariffs/%s/standard-unit-rates/", c.baseURI, productCode, tariffCode), &res); err != nil {
		return res, fmt.Errorf("standard unit rates: %w", err)
	}
	if len(res.Results) > 0 {
		return res, nil
	}

	var product product
	if err := c.getter.GetJSON(fmt.Sprintf("%s/products/%s/", c.baseURI, productCode), &product); err != nil {
		return res, fmt.Errorf("product: %w", err)
	}

	links := product.rateLinks(tariffCode)
	day, dayOK := links[rateRelationDay]
	night, nightOK := links[rateRelationNight]
	if !dayOK || !nightOK {
		return res, nil
	}

	dayRates, err := c.getUnitRates(day)
	if err != nil {
		return res, fmt.Errorf("day unit rates: %w", err)
	}
	nightRates, err := c.getUnitRates(night)
	if err != nil {
		return res, fmt.Errorf("night unit rates: %w", err)
	}

	location, err := time.LoadLocation("Europe/London")
	if err != nil {
		return res, fmt.Errorf("load Octopus tariff location: %w", err)
	}

	res.Results = dayNightRates(dayRates.Results, nightRates.Results, now, location)
	res.Count = uint64(len(res.Results))
	return res, nil
}

func (c *Client) getUnitRates(link string) (UnitRates, error) {
	base, err := url.Parse(c.baseURI + "/")
	if err != nil {
		return UnitRates{}, fmt.Errorf("parse base URI: %w", err)
	}
	reference, err := url.Parse(link)
	if err != nil {
		return UnitRates{}, fmt.Errorf("parse rate URI: %w", err)
	}

	var res UnitRates
	err = c.getter.GetJSON(base.ResolveReference(reference).String(), &res)
	return res, err
}

func dayNightRates(dayRates, nightRates []Rate, now time.Time, location *time.Location) []Rate {
	start := now.Truncate(rateSlotDuration)
	end := start.Add(forecastDuration)
	res := make([]Rate, 0, int(forecastDuration/rateSlotDuration))

	for slot := start; slot.Before(end); slot = slot.Add(rateSlotDuration) {
		rates := dayRates
		local := slot.In(location)
		minutes := local.Hour()*60 + local.Minute()
		// Intelligent Octopus Go applies night rates from 23:30 to 05:30 UK local time.
		if minutes >= 23*60+30 || minutes < 5*60+30 {
			rates = nightRates
		}

		if rate, ok := rateAt(rates, slot); ok {
			rate.ValidityStart = slot
			rate.ValidityEnd = slot.Add(rateSlotDuration)
			res = append(res, rate)
		}
	}

	return res
}

func rateAt(rates []Rate, slot time.Time) (Rate, bool) {
	var (
		res   Rate
		found bool
	)
	for _, rate := range rates {
		if slot.Before(rate.ValidityStart) || (!rate.ValidityEnd.IsZero() && !slot.Before(rate.ValidityEnd)) {
			continue
		}
		if !found || rate.ValidityStart.After(res.ValidityStart) {
			res = rate
			found = true
		}
	}
	return res, found
}

// ProductURI defines the location of the tariff information page. Substitute %s with tariff name.
const ProductURI = "https://api.octopus.energy/v1/products/%s/"

// RatesURI defines the location of the full tariff rates page, including speculation.
// Substitute first %s with product code, second with tariff code.
const RatesURI = ProductURI + "electricity-tariffs/%s/standard-unit-rates/"

// ConstructRatesAPIFromProductAndRegionCode returns a validly formatted, fully qualified URI to the unit rate information
// derived from the given product code and region.
func ConstructRatesAPIFromProductAndRegionCode(product string, region string) string {
	tCode := TariffCodeFromProductAndRegionCode(product, region)
	return fmt.Sprintf(RatesURI, product, tCode)
}

// TariffCodeFromProductAndRegionCode constructs an import tariff code.
func TariffCodeFromProductAndRegionCode(product, region string) string {
	return strings.ToUpper(fmt.Sprintf("E-1R-%s-%s", product, region))
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
	pCode := ProductCodeFromTariffCode(tariff)
	return fmt.Sprintf(RatesURI, pCode, tariff)
}

// ProductCodeFromTariffCode extracts the product code from a tariff code.
func ProductCodeFromTariffCode(tariff string) string {
	if len(tariff) < 7 {
		return ""
	}
	return tariff[5 : len(tariff)-2]
}

type UnitRates struct {
	Count    uint64 `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []Rate `json:"results"`
}

const (
	rateRelationDay   = "day_unit_rates"
	rateRelationNight = "night_unit_rates"
)

type product struct {
	SingleRegisterElectricityTariffs map[string]map[string]tariff `json:"single_register_electricity_tariffs"`
	DualRegisterElectricityTariffs   map[string]map[string]tariff `json:"dual_register_electricity_tariffs"`
	FourRateEVElectricityTariffs     map[string]map[string]tariff `json:"four_rate_ev_electricity_tariffs"`
}

type tariff struct {
	Code  string `json:"code"`
	Links []link `json:"links"`
}

type link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

func (p product) rateLinks(tariffCode string) map[string]string {
	res := make(map[string]string)
	groups := []map[string]map[string]tariff{
		p.SingleRegisterElectricityTariffs,
		p.DualRegisterElectricityTariffs,
		p.FourRateEVElectricityTariffs,
	}
	for _, group := range groups {
		for _, paymentMethods := range group {
			for _, tariff := range paymentMethods {
				if tariff.Code != tariffCode {
					continue
				}
				for _, link := range tariff.Links {
					res[link.Rel] = link.Href
				}
				return res
			}
		}
	}
	return res
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
