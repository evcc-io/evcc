package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
)

// OnboardingItems are the setup checklist rows a user can dismiss
var OnboardingItems = []string{"battery", "pv", "grid", "loadpoint", "heater", "consumer", "gridImport", "gridExport", "co2", "solar"}

type onboardingState struct {
	Done      bool     `json:"done"`
	Dismissed []string `json:"dismissed"`
}

// OnboardingSettings returns the persisted setup wizard and checklist state
func OnboardingSettings() onboardingState {
	var res onboardingState
	_ = settings.Json(keys.Onboarding, &res)
	if res.Dismissed == nil {
		res.Dismissed = []string{}
	}
	return res
}

func updateOnboardingHandler(pub publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req onboardingState
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		if req.Dismissed == nil {
			req.Dismissed = []string{}
		}
		for _, item := range req.Dismissed {
			if !slices.Contains(OnboardingItems, item) {
				jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid item: %s", item))
				return
			}
		}

		if err := settings.SetJson(keys.Onboarding, req); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		pub(keys.Onboarding, req)
		jsonWrite(w, req)
	}
}
