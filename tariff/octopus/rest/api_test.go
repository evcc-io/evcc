package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChargeCapRates(t *testing.T) {
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
						{"href": "/day-unit-rates/", "method": "GET", "rel": "day_unit_rates"},
						{"href": "/night-unit-rates/", "method": "GET", "rel": "night_unit_rates"}
					]
				}}
			}
		}`))
	})
	mux.HandleFunc("/day-unit-rates/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":1,"next":null,"previous":null,"results":[{"value_inc_vat":30.371355,"valid_from":"2026-07-05T23:00:00Z","valid_to":null,"payment_method":null}]}`))
	})
	mux.HandleFunc("/night-unit-rates/", func(w http.ResponseWriter, _ *http.Request) {
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

	assert.Equal(t, 30.371355, rates.Results[0].PriceInclusiveTax)
	assert.Equal(t, 6.89997, rates.Results[23].PriceInclusiveTax)
	assert.Equal(t, 6.89997, rates.Results[34].PriceInclusiveTax)
	assert.Equal(t, 30.371355, rates.Results[35].PriceInclusiveTax)
	assert.Equal(t, now, rates.Results[0].ValidityStart)
	assert.Equal(t, now.Add(48*time.Hour), rates.Results[len(rates.Results)-1].ValidityEnd)
}

func TestStandardUnitRates(t *testing.T) {
	const (
		productCode = "AGILE-24-10-01"
		tariffCode  = "E-1R-AGILE-24-10-01-H"
	)

	mux := http.NewServeMux()
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
}
