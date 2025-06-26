package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/glebarez/sqlite"
)

func updateSponsortokenHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	if req.Token != "" {
		if err := sponsor.ConfigureSponsorship(req.Token); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
	}

	// TODO find better place
	settings.SetString(keys.SponsorToken, req.Token)
	setConfigDirty()

	jsonResult(w, sponsor.Status())
}

func deleteSponsorTokenHandler(w http.ResponseWriter, r *http.Request) {
	settings.SetString(keys.SponsorToken, "")
	setConfigDirty()
	jsonResult(w, true)
}

func getBackup(authObject auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !authObject.IsAdminPasswordValid(req.Password) {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		settings.Persist()

		dial, ok := db.Instance.Dialector.(*sqlite.Dialector)
		if !ok {
			http.Error(w, "DB is not sqlite", http.StatusInternalServerError)
			return
		}

		path := strings.SplitN(dial.DSN, "?", 2)[0]

		f, err := os.Open(path)
		if err != nil {
			http.Error(w, "Could not open DB file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="evcc-backup.db"`)

		if _, err := io.Copy(w, f); err != nil {
			http.Error(w, "Error streaming DB file: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
