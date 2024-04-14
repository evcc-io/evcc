package server

import (
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/gorilla/mux"
)

type sponsorResult struct {
	IsAuthorized       bool      `json:"isAuthorized"`
	IsAuthorizedForApi bool      `json:"isAuthorizedForApi"`
	Subject            string    `json:"subject"`
	ExpiresAt          time.Time `json:"expires"`
}

func sponsorStatus() sponsorResult {
	return sponsorResult{
		IsAuthorized:       sponsor.IsAuthorized(),
		IsAuthorizedForApi: sponsor.IsAuthorizedForApi(),
		Subject:            sponsor.Subject,
		ExpiresAt:          sponsor.ExpiresAt,
	}
}

func sponsorStatusHandler(w http.ResponseWriter, r *http.Request) {
	jsonResult(w, sponsorStatus())
}

func updateSponsortokenHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	if err := sponsor.ConfigureSponsorship(token); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	jsonResult(w, sponsorStatus())
}
