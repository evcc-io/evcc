package rest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFourRateBaseRates(t *testing.T) {
	const (
		productCode = "IOG-SMB-VAR-24-10-29"
		tariffCode  = "E-1R-IOG-SMB-VAR-24-10-29-H"
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/products/"+productCode+"/electricity-tariffs/"+tariffCode+"/standard-unit-rates/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":0,"next":null,"previous":null,"results":[]}`))
	})
	mux.HandleFunc("/products/"+productCode+"/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"four_rate_ev_electricity_tariffs": {
				"_H": {"direct_debit_monthly": {
					"code": "E-1R-IOG-SMB-VAR-24-10-29-H",
					"links": [
						{"href": "/products/IOG-SMB-VAR-24-10-29/electricity-tariffs/E-1R-IOG-SMB-VAR-24-10-29-H/day-unit-rates/", "method": "GET", "rel": "day_unit_rates"},
						{"href": "/products/IOG-SMB-VAR-24-10-29/electricity-tariffs/E-1R-IOG-SMB-VAR-24-10-29-H/night-unit-rates/", "method": "GET", "rel": "night_unit_rates"}
					]
				}}
			}
		}`))
	})
	mux.HandleFunc("/products/"+productCode+"/electricity-tariffs/"+tariffCode+"/day-unit-rates/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":1,"next":null,"previous":null,"results":[{"value_inc_vat":30.371355,"valid_from":"2026-07-05T23:00:00Z","valid_to":null,"payment_method":null}]}`))
	})
	mux.HandleFunc("/products/"+productCode+"/electricity-tariffs/"+tariffCode+"/night-unit-rates/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":1,"next":null,"previous":null,"results":[{"value_inc_vat":6.89997,"valid_from":"2026-07-05T23:00:00Z","valid_to":null,"payment_method":null}]}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	helper := request.NewHelper(util.NewLogger("octopus-test"))
	client := newClient(helper, server.URL)
	now := time.Date(2026, time.September, 11, 11, 0, 0, 0, time.UTC)

	rates, err := client.UnitRates(productCode, tariffCode, now)
	require.NoError(t, err)
	require.Len(t, rates.Results, 96)
	assert.True(t, rates.BaseRates)

	assert.Equal(t, 30.371355, rates.Results[0].PriceInclusiveTax)
	assert.Equal(t, 6.89997, rates.Results[23].PriceInclusiveTax)
	assert.Equal(t, 6.89997, rates.Results[34].PriceInclusiveTax)
	assert.Equal(t, 30.371355, rates.Results[35].PriceInclusiveTax)
	assert.Equal(t, now, rates.Results[0].ValidityStart)
	assert.Equal(t, now.Add(48*time.Hour), rates.Results[len(rates.Results)-1].ValidityEnd)
}

func TestFourRateWithoutStandardEndpoint(t *testing.T) {
	const (
		productCode = "IOG-SMB-VAR-24-10-29"
		tariffCode  = "E-1R-IOG-SMB-VAR-24-10-29-H"
	)
	ratePath := "/products/" + productCode + "/electricity-tariffs/" + tariffCode + "/"
	mux := http.NewServeMux()
	mux.HandleFunc(ratePath+"standard-unit-rates/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/products/"+productCode+"/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"four_rate_ev_electricity_tariffs":{"_H":{"direct_debit_monthly":{"code":%q,"links":[{"href":%q,"rel":"day_unit_rates"},{"href":%q,"rel":"night_unit_rates"}]}}}}`, tariffCode, ratePath+"day-unit-rates/", ratePath+"night-unit-rates/")
	})
	for _, path := range []string{ratePath + "day-unit-rates/", ratePath + "night-unit-rates/"} {
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"results":[{"value_inc_vat":12.34,"valid_from":"2026-07-05T23:00:00Z","valid_to":null}]}`))
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := newClient(request.NewHelper(util.NewLogger("octopus-test")), server.URL)
	now := time.Date(2026, time.September, 11, 11, 0, 0, 0, time.UTC)
	rates, err := client.UnitRates(productCode, tariffCode, now)
	require.NoError(t, err)
	require.Len(t, rates.Results, 96)
}

func TestStandardUnitRates(t *testing.T) {
	const (
		productCode = "AGILE-24-10-01"
		tariffCode  = "E-1R-AGILE-24-10-01-H"
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/products/"+productCode+"/", func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("standard-rate product should not request product metadata")
	})
	mux.HandleFunc("/products/"+productCode+"/electricity-tariffs/"+tariffCode+"/standard-unit-rates/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":1,"next":null,"previous":null,"results":[{"value_inc_vat":12.34,"valid_from":"2026-09-11T11:00:00Z","valid_to":"2026-09-11T11:30:00Z","payment_method":null}]}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	helper := request.NewHelper(util.NewLogger("octopus-test"))
	client := newClient(helper, server.URL)
	rates, err := client.UnitRates(productCode, tariffCode, time.Time{})

	require.NoError(t, err)
	require.Len(t, rates.Results, 1)
	assert.Equal(t, 12.34, rates.Results[0].PriceInclusiveTax)
	assert.False(t, rates.BaseRates)
}

func TestFourRatePaymentMethods(t *testing.T) {
	now := time.Date(2026, time.September, 11, 11, 0, 0, 0, time.UTC)
	location, err := time.LoadLocation("Europe/London")
	require.NoError(t, err)

	rates := []Rate{
		{ValidityStart: now.Add(-time.Hour), PriceInclusiveTax: 20, PaymentMethod: RatePaymentMethodDirectDebit},
		{ValidityStart: now.Add(-time.Hour), PriceInclusiveTax: 25, PaymentMethod: RatePaymentMethodNotDirectDebit},
	}
	result := dayNightRates(rates, rates, now, location)

	require.Len(t, result, 192)
	for _, method := range []string{RatePaymentMethodDirectDebit, RatePaymentMethodNotDirectDebit} {
		count := 0
		for _, rate := range result {
			if rate.PaymentMethod == method {
				count++
			}
		}
		assert.Equal(t, 96, count, method)
	}
}

func TestUnitRatesRejectsIncompleteFourRate(t *testing.T) {
	const (
		productCode = "IOG-SMB-VAR-24-10-29"
		tariffCode  = "E-1R-IOG-SMB-VAR-24-10-29-H"
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/products/"+productCode+"/electricity-tariffs/"+tariffCode+"/standard-unit-rates/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	mux.HandleFunc("/products/"+productCode+"/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"four_rate_ev_electricity_tariffs":{"_H":{"direct_debit_monthly":{"code":%q,"links":[{"href":"/day-unit-rates/","rel":"day_unit_rates"}]}}}}`, tariffCode)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := newClient(request.NewHelper(util.NewLogger("octopus-test")), server.URL)
	_, err := client.UnitRates(productCode, tariffCode, time.Now())
	require.ErrorContains(t, err, "night")
}

func TestUnitRatesRejectsEconomySeven(t *testing.T) {
	const (
		productCode = "VAR-22-11-01"
		tariffCode  = "E-2R-VAR-22-11-01-H"
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/products/"+productCode+"/electricity-tariffs/"+tariffCode+"/standard-unit-rates/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	mux.HandleFunc("/products/"+productCode+"/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"dual_register_electricity_tariffs":{"_H":{"direct_debit_monthly":{"code":%q,"links":[{"href":"/day-unit-rates/","rel":"day_unit_rates"},{"href":"/night-unit-rates/","rel":"night_unit_rates"}]}}}}`, tariffCode)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := newClient(request.NewHelper(util.NewLogger("octopus-test")), server.URL)
	_, err := client.UnitRates(productCode, tariffCode, time.Now())
	require.ErrorContains(t, err, "no standard unit rates")
}

func TestUnitRatesRejectsExternalRateLink(t *testing.T) {
	client := newClient(request.NewHelper(util.NewLogger("octopus-test")), "https://api.octopus.energy/v1")
	_, err := client.getUnitRates("http://127.0.0.1/day-unit-rates/", "https://api.octopus.energy/v1/products/IOG/electricity-tariffs/E-1R-IOG-H/day-unit-rates/", time.Now())
	require.ErrorContains(t, err, "rate URI")
}

func TestUnitRatesFollowsPages(t *testing.T) {
	const path = "/products/IOG-SMB-VAR-24-10-29/electricity-tariffs/E-1R-IOG-SMB-VAR-24-10-29-H/day-unit-rates/"
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.URL.Query().Get("period_from"))
		assert.NotEmpty(t, r.URL.Query().Get("period_to"))
		if r.URL.Query().Get("page") == "2" {
			_, _ = w.Write([]byte(`{"next":null,"results":[{"value_inc_vat":25,"valid_from":"2026-09-11T11:00:00Z","valid_to":null}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"next":"?page=2","results":[{"value_inc_vat":20,"valid_from":"2026-09-10T11:00:00Z","valid_to":"2026-09-11T11:00:00Z"}]}`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := newClient(request.NewHelper(util.NewLogger("octopus-test")), server.URL)
	uri := server.URL + path
	rates, err := client.getUnitRates(uri, uri, time.Date(2026, time.September, 11, 11, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, rates.Results, 2)
}
