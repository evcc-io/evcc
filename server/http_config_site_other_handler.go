package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/sponsor"
)

func setExperimental(pub publisher) func(bool) error {
	return func(b bool) error {
		settings.SetBool(keys.Experimental, b)
		pub(keys.Experimental, b)
		return nil
	}
}

func getExperimental() bool {
	b, _ := settings.Bool(keys.Experimental)
	return b
}

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
				Status:     sponsor.RedactedStatus(),
				YamlSource: globalconfig.YamlSourceNone,
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
			Status:     sponsor.Status{},
			YamlSource: globalconfig.YamlSourceNone,
		})

		setConfigDirty()
		jsonWrite(w, true)
	}
}
