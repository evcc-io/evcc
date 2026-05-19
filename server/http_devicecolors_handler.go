package server

import (
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"regexp"

	"github.com/evcc-io/evcc/core/colors"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/site"
)

var hexColorRE = regexp.MustCompile(`^#[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$`)

// updateDeviceColor sets/removes a single title→hex association.
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
		color := req.Color
		if color != "" {
			if !hexColorRE.MatchString(color) {
				jsonError(w, http.StatusBadRequest, errors.New("invalid hex color"))
				return
			}
			if len(color) == 7 {
				color += "FF" // normalize to 8-digit
			}
		}

		m := colors.Get()
		m[req.Title] = color

		// clone — Publish serializes async
		site.Publish(keys.DeviceColors, maps.Clone(m))

		// delete only after publish to enforce update in client
		if color == "" {
			delete(m, req.Title)
		}
		if err := colors.Save(m); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		jsonWrite(w, m)
	}
}
