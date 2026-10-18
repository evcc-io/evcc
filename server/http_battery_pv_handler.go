package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type batteryPVResolver interface {
	ResolveBatteryPVControl(string, string, string) error
}

func batteryPVDecisionHandler(site batteryPVResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Battery  string `json:"battery"`
			Action   string `json:"action"`
			Revision string `json:"revision"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("one JSON object required"))
			return
		}
		if body.Battery == "" || body.Revision == "" || (body.Action != "disable" && body.Action != "recover") {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("battery, revision and valid action required"))
			return
		}
		if err := site.ResolveBatteryPVControl(body.Battery, body.Action, body.Revision); err != nil {
			jsonError(w, http.StatusConflict, err)
			return
		}
		jsonWrite(w, map[string]bool{"accepted": true})
	}
}
