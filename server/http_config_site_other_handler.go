package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/sponsor"
)

func updateSponsortokenHandler(pub publisher) func(w http.ResponseWriter, r *http.Request) {
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

			pub(keys.Sponsor, globalconfig.ConfigStatus{
				Status:   sponsor.RedactedStatus(),
				FromYaml: false,
			})
		}

		// TODO find better place
		settings.SetString(keys.SponsorToken, req.Token)
		setConfigDirty()

		jsonWrite(w, sponsor.RedactedStatus())
	}
}

func deleteSponsorTokenHandler(pub publisher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		settings.SetString(keys.SponsorToken, "")

		pub(keys.Sponsor, globalconfig.ConfigStatus{
			Status:   sponsor.Status{},
			FromYaml: false,
		})

		setConfigDirty()
		jsonWrite(w, true)
	}
}
