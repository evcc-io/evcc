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

func restoreDatabase(shutdown func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse multipart form
		err := r.ParseMultipartForm(32 << 20) // 32MB max memory
		if err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get uploaded file: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Get DB path
		dial, ok := db.Instance.Dialector.(*sqlite.Dialector)
		if !ok {
			http.Error(w, "DB is not sqlite", http.StatusInternalServerError)
			return
		}
		path := strings.SplitN(dial.DSN, "?", 2)[0]

		settings.Persist()

		// Overwrite DB file
		f, err := os.Create(path)
		if err != nil {
			http.Error(w, "Could not open DB file for writing: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		if _, err := io.Copy(f, file); err != nil {
			http.Error(w, "Failed to write DB file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		shutdown()
		w.WriteHeader(http.StatusNoContent)
	}
}

func resetDatabase(shutdown func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Sessions bool `json:"sessions"`
			Settings bool `json:"settings"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if req.Sessions {
			query := db.Instance.Exec("DELETE FROM sessions")
			if query.Error != nil {
				jsonError(w, http.StatusInternalServerError, query.Error)
				return
			}
		}

		if req.Settings {
			query := db.Instance.Exec("DELETE FROM settings")
			if query.Error != nil {
				jsonError(w, http.StatusInternalServerError, query.Error)
				return
			}
			query = db.Instance.Exec("DELETE FROM configs")
			if query.Error != nil {
				jsonError(w, http.StatusInternalServerError, query.Error)
				return
			}
		}

		shutdown()
		jsonResult(w, true)
	}
}
