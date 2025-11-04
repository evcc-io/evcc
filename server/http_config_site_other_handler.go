package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/util/sponsor"
)

func updateSponsortokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token string `json:"token"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := sponsor.SaveToken(req.Token, setConfigDirty); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, sponsor.Status())
	}
}

func deleteSponsorTokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sponsor.DeleteToken(setConfigDirty)
		jsonWrite(w, true)
	}
}
