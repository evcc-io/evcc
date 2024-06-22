package server

import (
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
)

type loadpointStaticConfig struct {
	// static config
	Charger        *string `json:"charger,omitempty"`
	Meter          *string `json:"meter,omitempty"`
	Circuit        *string `json:"circuit,omitempty"`
	DefaultVehicle *string `json:"defaultVehicle,omitempty"`
	Title          string  `json:"title"`
}

func getLoadpointStaticConfig(lp loadpoint.API) loadpointStaticConfig {
	return loadpointStaticConfig{
		Charger:        util.PtrTo(lp.GetChargerName()),
		Meter:          util.PtrTo(lp.GetMeterName()),
		Circuit:        util.PtrTo(lp.GetCircuitName()),
		DefaultVehicle: util.PtrTo(lp.GetDefaultVehicle()),
		Title:          lp.GetTitle(),
	}
}

type loadpointFullConfig struct {
	ID   int    `json:"id,omitempty"` // db row id
	Name string `json:"name"`         // either slice index (yaml) or db:<row id>

	// static config
	loadpointStaticConfig

	// dynamic config
	Mode           string  `json:"mode"`
	Priority       int     `json:"priority"`
	Phases         int     `json:"phases"`
	MinCurrent     float64 `json:"minCurrent"`
	MaxCurrent     float64 `json:"maxCurrent"`
	SmartCostLimit float64 `json:"smartCostLimit"`

	Thresholds loadpoint.ThresholdsConfig `json:"thresholds"`
	Soc        loadpoint.SocConfig        `json:"soc"`
}

// loadpointConfig returns a single loadpoint's configuration
func loadpointConfig(id int, lp loadpoint.API) loadpointFullConfig {
	res := loadpointFullConfig{
		ID: id,

		loadpointStaticConfig: getLoadpointStaticConfig(lp),

		Mode:           string(lp.GetMode()),
		Priority:       lp.GetPriority(),
		Phases:         lp.GetPhases(),
		MinCurrent:     lp.GetMinCurrent(),
		MaxCurrent:     lp.GetMaxCurrent(),
		SmartCostLimit: lp.GetSmartCostLimit(),
		Thresholds:     lp.GetThresholds(),
		Soc:            lp.GetSocConfig(),
	}

	return res
}

// loadpointsConfigHandler returns a device configurations by class
func loadpointsConfigHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var res []loadpointFullConfig

		for id, lp := range site.Loadpoints() {
			res = append(res, loadpointConfig(id, lp))
		}

		jsonResult(w, res)
	}
}

// loadpointConfigHandler returns a device configurations by class
func loadpointConfigHandler(id int, lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := loadpointConfig(id, lp)

		jsonResult(w, res)
	}
}

// newLoadpointHandler creates a new loadpoint
func newLoadpointHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload loadpointFullConfig

		if err := jsonDecoder(r.Body).Decode(&payload); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
}

// deleteLoadpointHandler deletes a loadpoint
func deleteLoadpointHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
}

// updateLoadpointHandler returns a device configurations by class
func updateLoadpointHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload loadpointFullConfig

		if err := jsonDecoder(r.Body).Decode(&payload); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// TODO: handle charger, meter, circuit, defaultVehicle, title

		var err error
		if err == nil {
			lp.SetPriority(payload.Priority)
		}

		if err == nil {
			lp.SetSmartCostLimit(payload.SmartCostLimit)
		}

		if err == nil {
			lp.SetThresholds(payload.Thresholds)
		}

		// TODO mode warning
		if err == nil {
			lp.SetSocConfig(payload.Soc)
		}

		var mode api.ChargeMode
		mode, err = api.ChargeModeString(payload.Mode)
		if err == nil {
			lp.SetMode(mode)
		}

		if err == nil {
			err = lp.SetPhases(payload.Phases)
		}

		if err == nil {
			err = lp.SetMinCurrent(payload.MinCurrent)
		}

		if err == nil {
			err = lp.SetMaxCurrent(payload.MaxCurrent)
		}

		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// TODO dirty handling
		w.WriteHeader(http.StatusOK)
	}
}
