package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
)

func updateSponsortokenHandler(valueChan chan<- util.Param) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token string `json:"token"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if req.Token != "" {
			if err := sponsor.ConfigureSponsorship(req.Token); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}
			sponsor.SetFromYaml(false)
		}

		// TODO find better place
		settings.SetString(keys.SponsorToken, req.Token)
		setConfigDirty()

		status := sponsor.Status()
		valueChan <- util.Param{Key: keys.Sponsor, Val: status}

		jsonWrite(w, status)
	}
}

func deleteSponsorTokenHandler(valueChan chan<- util.Param) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings.SetString(keys.SponsorToken, "")
		setConfigDirty()

		status := sponsor.Status()
		valueChan <- util.Param{Key: keys.Sponsor, Val: status}

		jsonWrite(w, true)
	}
}
