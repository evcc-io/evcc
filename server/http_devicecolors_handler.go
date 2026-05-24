package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/ui"
)

var hexColorRE = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func updateDeviceColor(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Title string `json:"title"`
			Color string `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		if req.Title == "" {
			jsonError(w, http.StatusBadRequest, errors.New("title required"))
			return
		}
		if req.Color != "" && !hexColorRE.MatchString(req.Color) {
			jsonError(w, http.StatusBadRequest, errors.New("invalid hex color"))
			return
		}

		m := ui.GetDeviceColors()
		if req.Color == "" {
			delete(m, req.Title)
		} else {
			m[req.Title] = req.Color
		}
		if err := ui.SaveDeviceColors(m); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		site.Publish(keys.DeviceColors, ui.DeviceColorList())
		jsonWrite(w, m)
	}
}
