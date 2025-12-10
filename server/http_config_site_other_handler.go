package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/sponsor"
)

func updateSponsortokenHandler(pub site.Publisher) func(w http.ResponseWriter, r *http.Request) {
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

			pub.Publish(keys.Sponsor, struct {
				Status   sponsor.Status `json:"status"`
				FromYaml bool           `json:"fromYaml"`
			}{
				Status:   sponsor.GetStatus(),
				FromYaml: false,
			})
		}

		// TODO find better place
		settings.SetString(keys.SponsorToken, req.Token)
		setConfigDirty()

		jsonWrite(w, sponsor.GetStatus())
	}
}

func deleteSponsorTokenHandler(pub site.Publisher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		settings.SetString(keys.SponsorToken, "")

		pub.Publish(keys.Sponsor, struct {
			Status   sponsor.Status `json:"status"`
			FromYaml bool           `json:"fromYaml"`
		}{
			Status:   sponsor.Status{},
			FromYaml: false,
		})

		setConfigDirty()
		jsonWrite(w, true)
	}
}
