package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
)

// updateOcppReportHandler persists the OCPP report rules, restoring masked
// secrets from the stored rules by loadpoint title, and applies them at runtime.
func updateOcppReportHandler(w http.ResponseWriter, r *http.Request) {
	var rules []ocpp.ReportRule
	if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// stationId and idTag are mandatory - no more "evcc-<loadpoint>" /
	// "EVCC" fallbacks; an empty idTag would also silently and permanently
	// fail every Authorize/StartTransaction (ocpp-go validates it required)
	for _, rule := range rules {
		if rule.StationID == "" {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("%s: stationId is required", rule.LoadpointTitle))
			return
		}
		if rule.IdTag == "" {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("%s: idTag is required", rule.LoadpointTitle))
			return
		}
	}

	// restore masked secrets (password, caCert) from stored rules by loadpoint title
	var old []ocpp.ReportRule
	if err := settings.Json(keys.OcppReport, &old); err == nil {
		stored := make(map[string]ocpp.ReportRule, len(old))
		for _, o := range old {
			stored[o.LoadpointTitle] = o
		}
		for i := range rules {
			if o, ok := stored[rules[i].LoadpointTitle]; ok {
				if err := mergeMaskedAny(&o, &rules[i]); err != nil {
					jsonError(w, http.StatusInternalServerError, err)
					return
				}
			}
		}
	}

	if err := settings.SetJson(keys.OcppReport, rules); err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}
	ocpp.ApplyReportRules(rules)

	jsonWrite(w, true)
}
