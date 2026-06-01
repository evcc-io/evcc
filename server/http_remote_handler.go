package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/server/remote"
)

// remoteClientsHandler returns the list of remote tunnel clients.
func remoteClientsHandler(r *remote.Remote) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		jsonWrite(w, r.Clients())
	}
}

// createRemoteClientHandler creates a new tunnel client and returns the
// cleartext password (shown to the user only once).
func createRemoteClientHandler(r *remote.Remote) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Username  string `json:"username"`
			ExpiresIn int64  `json:"expiresIn"` // seconds; 0 = never
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		client, password, err := r.CreateClient(body.Username, time.Duration(body.ExpiresIn)*time.Second)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, struct {
			Username  string     `json:"username"`
			Password  string     `json:"password"`
			CreatedAt time.Time  `json:"createdAt"`
			ExpiresAt *time.Time `json:"expiresAt,omitempty"`
		}{
			Username:  client.Username,
			Password:  password,
			CreatedAt: client.CreatedAt,
			ExpiresAt: client.ExpiresAt,
		})
	}
}

// deleteRemoteClientHandler removes a tunnel client by username.
// Username is passed as a query parameter to allow arbitrary characters.
func deleteRemoteClientHandler(r *remote.Remote) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		username := req.URL.Query().Get("username")
		if err := r.DeleteClient(username); err != nil {
			jsonError(w, http.StatusNotFound, err)
			return
		}
		jsonWrite(w, true)
	}
}
