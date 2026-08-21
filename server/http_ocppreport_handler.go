package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
)

// updateOcppReportHandler persists the OCPP report rules, restoring masked
// secrets from the stored rules by loadpoint title, and applies them at runtime.
func updateOcppReportHandler(w http.ResponseWriter, r *http.Request) {
	var rules []ocpp.ReportRule
	if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
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
