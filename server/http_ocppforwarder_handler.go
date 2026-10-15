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
	// compare the stations the connections reach, an omitted station id being the charger's own
	primary := rule.ConnectionKey(rule.StationID)
	seen := make(map[string]bool, len(rule.Mirrors))

	for _, m := range rule.Mirrors {
		key := m.ConnectionKey(rule.StationID)

		switch {
		case m.UpstreamURL == "":
			return fmt.Errorf("%s: read-only upstream needs a URL", rule.StationID)
		case key == primary:
			return fmt.Errorf("%s: read-only upstream %s is the same connection as the main upstream", rule.StationID, m.UpstreamURL)
		case seen[key]:
			return fmt.Errorf("%s: read-only upstream %s is listed twice", rule.StationID, m.UpstreamURL)
		}
		seen[key] = true
	}

	return nil
}

// restoreMaskedMirrors fills masked secrets of the submitted mirrors from the stored ones.
func restoreMaskedMirrors(stored, submitted []ocpp.MirrorRule) {
	byKey := make(map[string]ocpp.MirrorRule, len(stored))
	byURL := make(map[string][]ocpp.MirrorRule, len(stored))
	for _, m := range stored {
		byKey[m.Key()] = m

		url := strings.TrimRight(m.UpstreamURL, "/")
		byURL[url] = append(byURL[url], m)
	}

	for i := range submitted {
		// editing the station id keeps the upstream, so its secrets still apply if it is unambiguous;
		// a mirror whose URL changed has none to restore, and the mask is never persisted
		o, ok := byKey[submitted[i].Key()]
		if !ok {
			if same := byURL[strings.TrimRight(submitted[i].UpstreamURL, "/")]; len(same) == 1 {
				o = same[0]
			}
		}

		if submitted[i].Password == masked {
			submitted[i].Password = o.Password
		}
		if submitted[i].CaCert == masked {
			submitted[i].CaCert = o.CaCert
		}
	}
}
