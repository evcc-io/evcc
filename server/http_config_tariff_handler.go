package server

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/config"
	"golang.org/x/text/currency"
)

// tariffsHandler returns assignment of tariff devices
func tariffsHandler(w http.ResponseWriter, r *http.Request) {
	var refs globalconfig.TariffRefs
	if settings.Exists(keys.TariffRefs) {
		if err := settings.Json(keys.TariffRefs, &refs); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
	}

	jsonWrite(w, refs)
}

func validateTariffRef(w http.ResponseWriter, ref string) bool {
	if ref == "" {
		return true
	}
	if _, err := config.Tariffs().ByName(ref); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

// updateTariffHandler updates tariff assignments
func updateTariffHandler(w http.ResponseWriter, r *http.Request) {
	var refs globalconfig.TariffRefs
	if err := jsonDecoder(r.Body).Decode(&refs); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// Validate all refs
	for _, ref := range append([]string{refs.Grid, refs.FeedIn, refs.Co2, refs.Planner}, refs.Solar...) {
		if !validateTariffRef(w, ref) {
			return
		}
	}

	// Save to settings
	if err := settings.SetJson(keys.TariffRefs, refs); err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	setConfigDirty()

	w.WriteHeader(http.StatusAccepted)
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
