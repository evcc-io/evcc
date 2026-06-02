package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
)

// updateOcppForwarderHandler persists the complete set of OCPP forwarder rules,
// restores masked secrets from the stored rules (matched by station id), applies
// the rules at runtime and triggers a republish of the forwarder config/status.
func updateOcppForwarderHandler(w http.ResponseWriter, r *http.Request) {
	var rules []ocpp.ForwarderRule
	if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// restore masked secrets (password, caCert) from stored rules by station id
	var old []ocpp.ForwarderRule
	if err := settings.Json(keys.OcppForwarder, &old); err == nil {
		stored := make(map[string]ocpp.ForwarderRule, len(old))
		for _, o := range old {
			stored[o.StationID] = o
		}
		for i := range rules {
			if o, ok := stored[rules[i].StationID]; ok {
				if err := mergeMaskedAny(&o, &rules[i]); err != nil {
					jsonError(w, http.StatusInternalServerError, err)
					return
				}
			}
		}
	}

	if err := settings.SetJson(keys.OcppForwarder, rules); err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}
	setConfigDirty()
	ocpp.ApplyForwarderRules(rules)

	jsonWrite(w, true)
}
