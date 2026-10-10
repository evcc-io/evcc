package server

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util/config"
	"golang.org/x/text/language"
)

// siteHandler returns a device configurations by class
func siteHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := struct {
			Title    string   `json:"title"`
			Grid     string   `json:"grid"`
			PV       []string `json:"pv"`
			Battery  []string `json:"battery"`
			Aux      []string `json:"aux"`
			Ext      []string `json:"ext"`
			Consumer []string `json:"consumer"`
			Curtail  []string `json:"curtail"`
		}{
			Title:    site.GetTitle(),
			Grid:     site.GetGridMeterRef(),
			Curtail:  site.GetCurtailerRefs(),
			PV:       site.GetPVMeterRefs(),
			Battery:  site.GetBatteryMeterRefs(),
			Aux:      site.GetAuxMeterRefs(),
			Ext:      site.GetExtMeterRefs(),
			Consumer: site.GetConsumerMeterRefs(),
		}

		jsonWrite(w, res)
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
			Title    *string
			Grid     *string
			PV       *[]string
			Battery  *[]string
			Aux      *[]string
			Ext      *[]string
			Consumer *[]string
			Curtail  *[]string
		}

		if err := jsonDecoder(r.Body).Decode(&payload); err != nil {
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

		if payload.Aux != nil {
			if !validateRefs(w, *payload.Aux) {
				return
			}

			site.SetAuxMeterRefs(*payload.Aux)
			setConfigDirty()
		}

		if payload.Ext != nil {
			if !validateRefs(w, *payload.Ext) {
				return
			}

			site.SetExtMeterRefs(*payload.Ext)
			setConfigDirty()
		}

		if payload.Consumer != nil {
			if !validateRefs(w, *payload.Consumer) {
				return
			}

			site.SetConsumerMeterRefs(*payload.Consumer)
			setConfigDirty()
		}

		if payload.Curtail != nil {
			for _, m := range *payload.Curtail {
				if _, err := config.Curtailers().ByName(m); err != nil {
					jsonError(w, http.StatusBadRequest, err)
					return
				}
			}

			site.SetCurtailerRefs(*payload.Curtail)
			setConfigDirty()
		}

		// persist immediately to keep meter refs consistent with device config on unclean shutdown
		if err := settings.Persist(); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		status := map[bool]int{false: http.StatusOK, true: http.StatusAccepted}
		w.WriteHeader(status[ConfigDirty()])
	}
}

// updateCountryHandler sets the site country as ISO 3166-1 alpha-2 code, empty string clears it
func updateCountryHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var val string
		if err := jsonDecoder(r.Body).Decode(&val); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if val != "" {
			region, err := language.ParseRegion(val)
			if err != nil || !region.IsCountry() || region.String() != val {
				jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid country code: %s", val))
				return
			}
		}

		site.SetCountry(val)

		w.WriteHeader(http.StatusOK)
	}
}
