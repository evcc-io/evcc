package server

import (
	"net/http"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
)

type loadpointStruct struct {
	ID         int      `json:"id"`
	Title      *string  `json:"title,omitempty"`
	Priority   *int     `json:"priority,omitempty"`
	Phases     *int     `json:"phases,omitempty"`
	MinCurrent *float64 `json:"minCurrent,omitempty"`
	MaxCurrent *float64 `json:"maxCurrent,omitempty"`
}

// loadpointConfig returns a single loadpoint's configuration
func loadpointConfig(id int, lp loadpoint.API) loadpointStruct {
	res := loadpointStruct{
		ID:         id,
		Title:      ptr(lp.GetTitle()),
		Priority:   ptr(lp.GetPriority()),
		Phases:     ptr(lp.GetPhases()),
		MinCurrent: ptr(lp.GetMinCurrent()),
		MaxCurrent: ptr(lp.GetMaxCurrent()),
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
func updateLoadpointHandler(id int, lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			payload loadpointStruct
			err     error
		)

		if err := jsonDecoder(r.Body).Decode(&payload); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if payload.Title != nil {
			lp.SetTitle(*payload.Title)
		}

		if payload.Priority != nil {
			lp.SetPriority(*payload.Priority)
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
