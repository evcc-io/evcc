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
	var standard UnitRates
	standardURI := fmt.Sprintf("%s/products/%s/electricity-tariffs/%s/standard-unit-rates/", c.baseURI, productCode, tariffCode)
	standardErr := c.getter.GetJSON(standardURI, &standard)
	if standardErr == nil && len(standard.Results) > 0 {
		return standard, nil
	}

	if !strings.HasPrefix(strings.ToUpper(productCode), "IOG-SMB-") {
		if standardErr != nil {
			return UnitRates{}, fmt.Errorf("standard unit rates: %w", standardErr)
		}
		return UnitRates{}, fmt.Errorf("no standard unit rates for tariff %s", tariffCode)
	}

	var product product
	if err := c.getter.GetJSON(fmt.Sprintf("%s/products/%s/", c.baseURI, productCode), &product); err != nil {
		return UnitRates{}, fmt.Errorf("product: %w", err)
	}
	links := product.rateLinks(tariffCode)
	if len(links) == 0 {
		return UnitRates{}, fmt.Errorf("no four-rate tariff links for %s", tariffCode)
	}
	day, dayOK := links[rateRelationDay]
	night, nightOK := links[rateRelationNight]
	if !dayOK {
		return UnitRates{}, fmt.Errorf("missing day unit rates for tariff %s", tariffCode)
	}
	if !nightOK {
		return UnitRates{}, fmt.Errorf("missing night unit rates for tariff %s", tariffCode)
	}

	rateURI := fmt.Sprintf("%s/products/%s/electricity-tariffs/%s/", c.baseURI, productCode, tariffCode)
	dayRates, err := c.getUnitRates(day, rateURI+"day-unit-rates/", now)
	if err != nil {
		return UnitRates{}, fmt.Errorf("day unit rates: %w", err)
	}
	nightRates, err := c.getUnitRates(night, rateURI+"night-unit-rates/", now)
	if err != nil {
		return UnitRates{}, fmt.Errorf("night unit rates: %w", err)
	}

	location, err := time.LoadLocation("Europe/London")
	if err != nil {
		return UnitRates{}, fmt.Errorf("load Octopus tariff location: %w", err)
	}

	var res UnitRates
	res.Results = dayNightRates(dayRates.Results, nightRates.Results, now, location)
	if len(res.Results) == 0 {
		return res, fmt.Errorf("no day or night rates for tariff %s", tariffCode)
	}
	res.Count = uint64(len(res.Results))
	res.BaseRates = true
	return res, nil
}

func (c *Client) getUnitRates(link, expectedURI string, now time.Time) (UnitRates, error) {
	base, err := url.Parse(c.baseURI + "/")
	if err != nil {
		return UnitRates{}, fmt.Errorf("parse base URI: %w", err)
	}
	reference, err := url.Parse(link)
	if err != nil {
		return UnitRates{}, fmt.Errorf("parse rate URI: %w", err)
	}

	resolved := base.ResolveReference(reference)
	if resolved.String() != expectedURI {
		return UnitRates{}, fmt.Errorf("unexpected rate URI: %s", resolved)
	}

	var res UnitRates
	start := now.Truncate(rateSlotDuration).UTC()
	end := start.Add(forecastDuration)
	query := resolved.Query()
	query.Set("period_from", start.Format(time.RFC3339))
	query.Set("period_to", end.Format(time.RFC3339))
	for page := 1; page <= 100; page++ {
		query.Set("page", fmt.Sprint(page))
		resolved.RawQuery = query.Encode()
		var batch UnitRates
		if err := c.getter.GetJSON(resolved.String(), &batch); err != nil {
			return UnitRates{}, fmt.Errorf("rate page %d: %w", page, err)
		}
		res.Results = append(res.Results, batch.Results...)
		if batch.Next == "" {
			res.Count = uint64(len(res.Results))
			return res, nil
		}
	}
	return UnitRates{}, fmt.Errorf("too many rate pages: %s", expectedURI)
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

		for _, rate := range ratesAt(rates, slot) {
			rate.ValidityStart = slot
			rate.ValidityEnd = slot.Add(rateSlotDuration)
			res = append(res, rate)
		}
	}

	return res
}

func ratesAt(rates []Rate, slot time.Time) []Rate {
	var res []Rate
	for _, rate := range rates {
		if slot.Before(rate.ValidityStart) || (!rate.ValidityEnd.IsZero() && !slot.Before(rate.ValidityEnd)) {
			continue
		}
		found := false
		for i := range res {
			if res[i].PaymentMethod != rate.PaymentMethod {
				continue
			}
			if rate.ValidityStart.After(res[i].ValidityStart) {
				res[i] = rate
			}
			found = true
			break
		}
		if !found {
			res = append(res, rate)
		}
	}
	return res
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
	Count     uint64 `json:"count"`
	Next      string `json:"next"`
	Previous  string `json:"previous"`
	Results   []Rate `json:"results"`
	BaseRates bool   `json:"-"`
}

const (
	rateRelationDay   = "day_unit_rates"
	rateRelationNight = "night_unit_rates"
)

type product struct {
	FourRateEVElectricityTariffs map[string]map[string]tariff `json:"four_rate_ev_electricity_tariffs"`
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
	for _, paymentMethods := range p.FourRateEVElectricityTariffs {
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
