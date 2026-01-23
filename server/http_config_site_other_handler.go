package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/sponsor"
)

var licenseKeyPattern = regexp.MustCompile(`^[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}$`)

func updateSponsortokenHandler(pub publisher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token string `json:"token"`
			Email string `json:"email"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		var instanceID string

		// License key activation flow
		if req.Email != "" {
			// Validate token matches license key pattern
			if !licenseKeyPattern.MatchString(strings.ToUpper(req.Token)) {
				jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid license key format"))
				return
			}

			// Activate license key
			var err error
			instanceID, err = sponsor.ActivateSponsorship(req.Token, req.Email)
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			settings.SetString(keys.SponsorInstanceId, instanceID)
		} else {
			// Load existing instance_id for JWT flow
			instanceID, _ = settings.String(keys.SponsorInstanceId)
		}

		if req.Token != "" {
			if err := sponsor.ConfigureSponsorship(req.Token, instanceID); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			pub(keys.Sponsor, struct {
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

func deleteSponsorTokenHandler(pub publisher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		settings.SetString(keys.SponsorToken, "")
		settings.SetString(keys.SponsorInstanceId, "")

		pub(keys.Sponsor, struct {
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
