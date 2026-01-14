package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/config"
)

// TariffRefs holds the tariff to usage type mapping
type TariffRefs struct {
	Currency string   `json:"currency"`
	Grid     string   `json:"grid"`
	FeedIn   string   `json:"feedin"`
	Co2      string   `json:"co2"`
	Planner  string   `json:"planner"`
	Solar    []string `json:"solar"`
}

// tariffsHandler returns current tariff assignments
func tariffsHandler(w http.ResponseWriter, r *http.Request) {
	var res TariffRefs
	if err := settings.Json(keys.TariffRefs, &res); err != nil {
		// return defaults if not configured
		res = TariffRefs{Currency: "EUR", Solar: []string{}}
	}
	if res.Solar == nil {
		res.Solar = []string{}
	}
	jsonWrite(w, res)
}

// updateTariffsHandler updates tariff assignments
func updateTariffsHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Currency *string
		Grid     *string
		FeedIn   *string
		Co2      *string
		Planner  *string
		Solar    *[]string
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// Load current assignments
	var assignments TariffRefs
	if err := settings.Json(keys.TariffRefs, &assignments); err != nil {
		assignments = TariffRefs{Currency: "EUR", Solar: []string{}}
	}

	// Update fields if provided
	if payload.Currency != nil {
		assignments.Currency = *payload.Currency
	}

	// Validate and update tariff references
	if payload.Grid != nil {
		if *payload.Grid != "" {
			if _, err := config.Tariffs().ByName(*payload.Grid); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}
		}
		assignments.Grid = *payload.Grid
	}

	if payload.FeedIn != nil {
		if *payload.FeedIn != "" {
			if _, err := config.Tariffs().ByName(*payload.FeedIn); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}
		}
		assignments.FeedIn = *payload.FeedIn
	}

	if payload.Co2 != nil {
		if *payload.Co2 != "" {
			if _, err := config.Tariffs().ByName(*payload.Co2); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}
		}
		assignments.Co2 = *payload.Co2
	}

	if payload.Planner != nil {
		if *payload.Planner != "" {
			if _, err := config.Tariffs().ByName(*payload.Planner); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}
		}
		assignments.Planner = *payload.Planner
	}

	if payload.Solar != nil {
		// Validate all solar tariff references
		for _, name := range *payload.Solar {
			if name != "" {
				if _, err := config.Tariffs().ByName(name); err != nil {
					jsonError(w, http.StatusBadRequest, err)
					return
				}
			}
		}
		assignments.Solar = *payload.Solar
	}

	// Save updated assignments
	settings.SetJson(keys.TariffRefs, assignments)
	setConfigDirty()

	status := map[bool]int{false: http.StatusOK, true: http.StatusAccepted}
	w.WriteHeader(status[ConfigDirty()])
}
