package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/config"
	"github.com/gorilla/mux"
	"golang.org/x/text/currency"
)

// tariffsHandler returns assignment of tariff devices
func tariffsHandler(w http.ResponseWriter, r *http.Request) {
	res := make(map[string]any)

	for _, usage := range api.TariffUsageValues() {
		value, _ := settings.String(usage.Key())

		if value == "" {
			continue
		}

		key := usage.String()

		if usage == api.TariffUsageSolar {
			res[key] = strings.Split(value, ",")
		} else {
			res[key] = value
		}
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

	// Parse tariff type string to TariffUsage enum
	usage, err := api.TariffUsageString(tariffType)
	if err != nil {
		jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid tariff type: %s", tariffType))
		return
	}

	// Handle solar (array) separately
	if usage == api.TariffUsageSolar {
		var existing []string
		if v, err := settings.String(usage.Key()); err == nil && v != "" {
			existing = strings.Split(v, ",")
		}
		settings.SetString(usage.Key(), strings.Join(append(existing, ref), ","))
	} else {
		settings.SetString(usage.Key(), ref)
	}

	setConfigDirty()

	status := map[bool]int{false: http.StatusOK, true: http.StatusAccepted}
	w.WriteHeader(status[ConfigDirty()])
}

// updateCurrencyHandler updates the currency setting
func updateCurrencyHandler(pub publisher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var val string
		if err := jsonDecoder(r.Body).Decode(&val); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		_, err := currency.ParseISO(val)
		if err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid currency code: %w", err))
			return
		}

		settings.SetString(keys.Currency, val)
		pub(keys.Currency, val)

		w.WriteHeader(http.StatusOK)
	}
}
