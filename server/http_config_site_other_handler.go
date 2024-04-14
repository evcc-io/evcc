package server

import (
	"net/http"

	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/gorilla/mux"
)

func sponsorStatusHandler(w http.ResponseWriter, r *http.Request) {
	jsonResult(w, sponsor.Status())
}

func updateSponsortokenHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	if err := sponsor.ConfigureSponsorship(token); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	jsonResult(w, sponsor.Status())
}
