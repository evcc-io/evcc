package server

import (
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
)

type loadpointStruct struct {
	ID             int      `json:"id"`
	Charger        *string  `json:"charger,omitempty"`
	Meter          *string  `json:"meter,omitempty"`
	DefaultVehicle *string  `json:"defaultVehicle,omitempty"`
	Title          *string  `json:"title,omitempty"`
	Mode           *string  `json:"mode,omitempty"`
	Priority       *int     `json:"priority,omitempty"`
	Phases         *int     `json:"phases,omitempty"`
	MinCurrent     *float64 `json:"minCurrent,omitempty"`
	MaxCurrent     *float64 `json:"maxCurrent,omitempty"`
	SmartCostLimit *float64 `json:"smartCostLimit,omitempty"`

	Thresholds *loadpoint.ThresholdsConfig `json:"thresholds,omitempty"`
	Soc        *loadpoint.SocConfig        `json:"soc,omitempty"`
}

// loadpointConfig returns a single loadpoint's configuration
func loadpointConfig(id int, lp loadpoint.API) loadpointStruct {
	res := loadpointStruct{
		ID:             id,
		Charger:        ptr(lp.GetCharger()),
		Meter:          ptr(lp.GetMeter()),
		DefaultVehicle: ptr(lp.GetDefaultVehicle()),
		Title:          ptr(lp.GetTitle()),
		Mode:           ptr(string(lp.GetMode())),
		Priority:       ptrZero(lp.GetPriority()),
		Phases:         ptrZero(lp.GetPhases()),
		MinCurrent:     ptr(lp.GetMinCurrent()),
		MaxCurrent:     ptr(lp.GetMaxCurrent()),
		SmartCostLimit: ptr(lp.GetSmartCostLimit()),
		Thresholds:     ptr(lp.GetThresholds()),
		Soc:            ptr(lp.GetSocConfig()),
	}

	return res
}

// loadpointsConfigHandler returns a device configurations by class
func loadpointsConfigHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var res []loadpointStruct

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

// updateLoadpointHandler returns a device configurations by class
func updateLoadpointHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload loadpointStruct

		if err := jsonDecoder(r.Body).Decode(&payload); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		var err error
		if payload.Charger != nil || payload.Meter != nil || payload.DefaultVehicle != nil {
			err = errors.New("not implemented")
		}

		if err == nil && payload.Title != nil {
			lp.SetTitle(*payload.Title)
		}

		if err == nil && payload.Priority != nil {
			lp.SetPriority(*payload.Priority)
		}

		if err == nil && payload.SmartCostLimit != nil {
			lp.SetSmartCostLimit(*payload.SmartCostLimit)
		}

		if err == nil && payload.Thresholds != nil {
			lp.SetThresholds(*payload.Thresholds)
		}

		// TODO mode warning
		if err == nil && payload.Soc != nil {
			lp.SetSocConfig(*payload.Soc)
		}

		if payload.Mode != nil {
			var mode api.ChargeMode
			mode, err = api.ChargeModeString(*payload.Mode)
			if err == nil {
				lp.SetMode(mode)
			}
		}

		if err == nil && payload.Phases != nil {
			err = lp.SetPhases(*payload.Phases)
		}

		if err == nil && payload.MinCurrent != nil {
			err = lp.SetMinCurrent(*payload.MinCurrent)
		}

		if err == nil && payload.MaxCurrent != nil {
			err = lp.SetMaxCurrent(*payload.MaxCurrent)
		}

		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// TODO dirty handling
		w.WriteHeader(http.StatusOK)
	}
}
