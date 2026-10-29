package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util/sponsor"
)

func setOptimizer(pub publisher) func(bool) error {
	return func(b bool) error {
		settings.SetBool(keys.Optimizer, b)
		pub(keys.Optimizer, b)
		if !b {
			// automatic mode cannot outlive the optimizer
			settings.SetString(keys.OptimizerAutomatic, core.OptimizerAutomaticOff)
			pub(keys.OptimizerAutomatic, core.OptimizerAutomaticOff)
		}
		return nil
	}
}

func getOptimizer() bool {
	b, _ := settings.Bool(keys.Optimizer)
	return b
}

func setOptimizerAutomatic(pub publisher, site *core.Site) func(string) error {
	return func(level string) error {
		if !slices.Contains(core.OptimizerAutomaticLevels, level) {
			return fmt.Errorf("invalid optimizer automatic level: %s", level)
		}

		settings.SetString(keys.OptimizerAutomatic, level)
		pub(keys.OptimizerAutomatic, level)

		// suggestions become control decisions, don't wait for the next slot
		site.Optimize()

		return nil
	}
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
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		token := req.Token

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
