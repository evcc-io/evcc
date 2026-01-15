package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/config"
	"github.com/gorilla/mux"
)

// tariffsHandler returns current tariff assignments
func tariffsHandler(w http.ResponseWriter, r *http.Request) {
	res := struct {
		Currency string   `json:"currency"`
		Grid     string   `json:"grid"`
		FeedIn   string   `json:"feedin"`
		Co2      string   `json:"co2"`
		Planner  string   `json:"planner"`
		Solar    []string `json:"solar"`
	}{
		Currency: "EUR",
		Solar:    []string{},
	}

	// Read individual keys
	if currency, _ := settings.String(keys.Currency); currency != "" {
		res.Currency = currency
	}
	res.Grid, _ = settings.String(keys.GridTariff)
	res.FeedIn, _ = settings.String(keys.FeedinTariff)
	res.Co2, _ = settings.String(keys.Co2Tariff)
	res.Planner, _ = settings.String(keys.PlannerTariff)

	if v, err := settings.String(keys.SolarTariffs); err == nil && v != "" {
		res.Solar = strings.Split(v, ",")
	} else {
		res.Solar = []string{}
	}

	jsonWrite(w, res)
}

func validateTariffRef(w http.ResponseWriter, ref string) bool {
	if ref != "" {
		if _, err := config.Tariffs().ByName(ref); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return false
		}
	}
	return true
}

// updateTariffHandler updates a specific tariff assignment by type
func updateTariffHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tariffType := vars["type"]

	var ref string
	if err := jsonDecoder(r.Body).Decode(&ref); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	if !validateTariffRef(w, ref) {
		return
	}

	switch tariffType {
	case "grid":
		settings.SetString(keys.GridTariff, ref)
		setConfigDirty()

	case "feedin":
		settings.SetString(keys.FeedinTariff, ref)
		setConfigDirty()

	case "co2":
		settings.SetString(keys.Co2Tariff, ref)
		setConfigDirty()

	case "planner":
		settings.SetString(keys.PlannerTariff, ref)
		setConfigDirty()

	case "solar":
		var existing []string
		if v, err := settings.String(keys.SolarTariffs); err == nil && v != "" {
			existing = strings.Split(v, ",")
		}
		settings.SetString(keys.SolarTariffs, strings.Join(append(existing, ref), ","))
		setConfigDirty()

	default:
		jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid tariff type: %s", tariffType))
		return
	}

	status := map[bool]int{false: http.StatusOK, true: http.StatusAccepted}
	w.WriteHeader(status[ConfigDirty()])
}

// updateCurrencyHandler updates the currency setting
func updateCurrencyHandler(w http.ResponseWriter, r *http.Request) {
	var currency string
	if err := jsonDecoder(r.Body).Decode(&currency); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	settings.SetString(keys.Currency, currency)

	w.WriteHeader(http.StatusOK)
}
