package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/sponsor"
)

var licenseKeyPattern = regexp.MustCompile(`^[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}$`)

func setOptimizer(pub publisher) func(bool) error {
	return func(b bool) error {
		settings.SetBool(keys.Optimizer, b)
		pub(keys.Optimizer, b)
		return nil
	}
}

func getOptimizer() bool {
	b, _ := settings.Bool(keys.Optimizer)
	return b
}

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
			Email string `json:"email"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		var token string

		// License key activation flow
		if req.Email != "" {
			// Validate token matches license key pattern
			if !licenseKeyPattern.MatchString(strings.ToUpper(req.Token)) {
				jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid license key format"))
				return
			}

			// Activate license key and receive JWT token
			var err error
			token, err = sponsor.ActivateSponsorship(req.Token, req.Email)
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}
		} else {
			// Use provided JWT token directly
			token = req.Token
		}

		if token != "" {
			if err := sponsor.ConfigureSponsorship(token); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			pub(keys.Sponsor, globalconfig.ConfigStatus{
				Status:     sponsor.RedactedStatus(),
				YamlSource: globalconfig.YamlSourceNone,
			})
		}

		// TODO find better place
		settings.SetString(keys.SponsorToken, token)
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
