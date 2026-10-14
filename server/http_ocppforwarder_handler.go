package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
)

// updateOcppForwarderHandler persists the OCPP forwarder rules, restoring masked
// secrets from the stored rules by station id, and applies them at runtime.
func updateOcppForwarderHandler(w http.ResponseWriter, r *http.Request) {
	var rules []ocpp.ForwarderRule
	if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	for _, rule := range rules {
		if err := validateMirrors(rule); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
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
				restoreMaskedMirrors(o.Mirrors, rules[i].Mirrors)

				// mirrors are restored above; merging would resurrect removed ones
				o.Mirrors = nil
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
	ocpp.ApplyForwarderRules(rules)

	jsonWrite(w, true)
}

// validateMirrors checks the read-only upstreams of a rule.
func validateMirrors(rule ocpp.ForwarderRule) error {
	seen := make(map[string]bool, len(rule.Mirrors))
	primary := strings.TrimRight(rule.UpstreamURL, "/") + "|" + rule.UpstreamStationID

	for _, m := range rule.Mirrors {
		switch {
		case m.UpstreamURL == "":
			return fmt.Errorf("%s: read-only upstream needs a URL", rule.StationID)
		case seen[m.Key()]:
			return fmt.Errorf("%s: read-only upstream %s is listed twice", rule.StationID, m.UpstreamURL)
		case m.Key() == primary:
			return fmt.Errorf("%s: read-only upstream %s is the same connection as the main upstream", rule.StationID, m.UpstreamURL)
		}
		seen[m.Key()] = true
	}

	return nil
}

// restoreMaskedMirrors fills masked secrets of the submitted mirrors from the stored ones.
func restoreMaskedMirrors(stored, submitted []ocpp.MirrorRule) {
	byKey := make(map[string]ocpp.MirrorRule, len(stored))
	for _, m := range stored {
		byKey[m.Key()] = m
	}

	for i := range submitted {
		// a mirror whose URL changed has no stored secrets to restore; never persist the mask
		o := byKey[submitted[i].Key()]
		if submitted[i].Password == masked {
			submitted[i].Password = o.Password
		}
		if submitted[i].CaCert == masked {
			submitted[i].CaCert = o.CaCert
		}
	}
}
