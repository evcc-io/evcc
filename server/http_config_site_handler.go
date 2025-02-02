package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util/config"
)

// siteHandler returns a device configurations by class
func siteHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := struct {
			Title   string   `json:"title"`
			Grid    string   `json:"grid"`
			PV      []string `json:"pv"`
			Battery []string `json:"battery"`
		}{
			Title:   site.GetTitle(),
			Grid:    site.GetGridMeterRef(),
			PV:      site.GetPVMeterRefs(),
			Battery: site.GetBatteryMeterRefs(),
		}

		jsonResult(w, res)
	}
}

func validateRefs(w http.ResponseWriter, refs []string) bool {
	for _, m := range refs {
		if _, err := config.Meters().ByName(m); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return false
		}
	}
	return true
}

// siteHandler returns a device configurations by class
func updateSiteHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Title   *string
			Grid    *string
			PV      *[]string
			Battery *[]string
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if payload.Title != nil {
			site.SetTitle(*payload.Title)
		}

		if payload.Grid != nil {
			if *payload.Grid != "" && !validateRefs(w, []string{*payload.Grid}) {
				return
			}

			site.SetGridMeterRef(*payload.Grid)
			setConfigDirty()
		}

		if payload.PV != nil {
			if !validateRefs(w, *payload.PV) {
				return
			}

			site.SetPVMeterRefs(*payload.PV)
			setConfigDirty()
		}

		if payload.Battery != nil {
			if !validateRefs(w, *payload.Battery) {
				return
			}

			site.SetBatteryMeterRefs(*payload.Battery)
			setConfigDirty()
		}

		status := map[bool]int{false: http.StatusOK, true: http.StatusAccepted}
		w.WriteHeader(status[ConfigDirty()])
	}
}
