package server

import (
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/gorilla/mux"
)

// authChallengeResponse is returned by the interactive auth endpoints. A nil
// challenge together with authenticated=true signals completion.
type authChallengeResponse struct {
	Authenticated bool               `json:"authenticated"`
	Challenge     *api.AuthChallenge `json:"challenge,omitempty"`
}

// authChallengeHandler returns the initial credential form for an interactive
// auth provider (e.g. email + password). The detailed error is logged by
// providerauth; the client only sees a stable message.
func authChallengeHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	challenge, err := providerauth.Challenge(id)
	if err != nil {
		jsonError(w, http.StatusBadRequest, errors.New("interactive login unavailable"))
		return
	}

	jsonWrite(w, authChallengeResponse{Challenge: challenge})
}

// authSubmitHandler processes user-provided values for an interactive auth
// provider, returning the next challenge (e.g. a captcha) or completion. The
// detailed error is logged by providerauth; the client only sees a stable
// message so backend internals are not exposed.
func authSubmitHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var values map[string]string
	if err := jsonDecoder(r.Body).Decode(&values); err != nil {
		jsonError(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}

	challenge, done, err := providerauth.Submit(id, values)
	if err != nil {
		jsonError(w, http.StatusBadRequest, errors.New("login failed - please check your details and try again"))
		return
	}

	if done {
		// devices using the provider are reinitialized on restart
		setConfigDirty()
	}

	jsonWrite(w, authChallengeResponse{Authenticated: done, Challenge: challenge})
}
