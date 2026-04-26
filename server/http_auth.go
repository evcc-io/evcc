package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util/auth"
	"github.com/gorilla/mux"
)

const authCookieName = "auth"

type updatePasswordRequest struct {
	Current string `json:"current"`
	New     string `json:"new"`
}

type loginRequest struct {
	Password string `json:"password"`
}

func updatePasswordHandler(authObject auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authObject.GetAuthMode() == auth.Locked {
			http.Error(w, "Forbidden in demo mode", http.StatusForbidden)
			return
		}

		var req updatePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// update password
		if authObject.IsAdminPasswordConfigured() {
			if !authObject.IsAdminPasswordValid(req.Current) {
				http.Error(w, "Invalid password", http.StatusBadRequest)
				return
			}

			if err := authObject.SetAdminPassword(req.New); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusAccepted)
			return
		}

		// create new password
		if err := authObject.SetAdminPassword(req.New); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// auto-login: set auth cookie
		if err := setAuthCookie(authObject, w); err != nil {
			http.Error(w, "Failed to generate JWT token.", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// apiKeyFromRequest returns the API key from the Authorization: Bearer header, or "" if absent
func apiKeyFromRequest(r *http.Request) string {
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return token
}

// jwtFromCookie returns the session JWT from the auth cookie, or "" if absent
func jwtFromCookie(r *http.Request) string {
	if cookie, _ := r.Cookie(authCookieName); cookie != nil {
		return cookie.Value
	}
	return ""
}

// validateAuth accepts a valid API key from the Authorization header, or a valid session JWT from the auth cookie
func validateAuth(authObject auth.Auth, r *http.Request) bool {
	if key := apiKeyFromRequest(r); key != "" {
		return authObject.ValidateApiKey(key)
	}
	return authObject.ValidateJwtToken(jwtFromCookie(r))
}

// requireAdminPassword passes when --disable-auth is set or the supplied password matches.
// Writes 401 and returns false otherwise.
func requireAdminPassword(w http.ResponseWriter, authObject auth.Auth, password string) bool {
	if authObject.GetAuthMode() == auth.Disabled || authObject.IsAdminPasswordValid(password) {
		return true
	}
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
	return false
}

// requireAdminPasswordOrApiKey is requireAdminPassword that also accepts a valid API-key
// Bearer header (for automation-friendly endpoints: backup, restore, reset).
func requireAdminPasswordOrApiKey(w http.ResponseWriter, r *http.Request, authObject auth.Auth, password string) bool {
	if key := apiKeyFromRequest(r); key != "" && authObject.ValidateApiKey(key) {
		return true
	}
	return requireAdminPassword(w, authObject, password)
}

// authStatusHandler login status (true/false) based on jwt token. Error if admin password is not configured
func authStatusHandler(authObject auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authObject.GetAuthMode() == auth.Disabled {
			w.Write([]byte("true"))
			return
		}

		if authObject.GetAuthMode() == auth.Locked {
			http.Error(w, "Forbidden in demo mode", http.StatusForbidden)
			return
		}

		if !authObject.IsAdminPasswordConfigured() {
			http.Error(w, "Not implemented", http.StatusNotImplemented)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if validateAuth(authObject, r) {
			w.Write([]byte("true"))
		} else {
			w.Write([]byte("false"))
		}
	}
}

func setAuthCookie(authObject auth.Auth, w http.ResponseWriter) error {
	lifetime := time.Hour * 24 * 90 // 90 day valid
	tokenString, err := authObject.GenerateJwtToken(lifetime)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(lifetime),
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func loginHandler(authObject auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authObject.GetAuthMode() == auth.Locked {
			http.Error(w, "Forbidden in demo mode", http.StatusForbidden)
			return
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !authObject.IsAdminPasswordValid(req.Password) {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		if err := setAuthCookie(authObject, w); err != nil {
			http.Error(w, "Failed to generate JWT token.", http.StatusInternalServerError)
			return
		}
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Path:     "/",
		HttpOnly: true,
	})
}

func ensureAuthHandler(authObject auth.Auth) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if next == nil {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}

			if authObject.GetAuthMode() == auth.Disabled {
				next.ServeHTTP(w, r)
				return
			}

			if authObject.GetAuthMode() == auth.Locked {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !validateAuth(authObject, r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func apiKeyStatusHandler(authObject auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonWrite(w, map[string]bool{"configured": authObject.IsApiKeyConfigured()})
	}
}

// regenerateApiKeyHandler creates or rotates the API key. Requires the admin password (a leaked API key cannot self-rotate)
func regenerateApiKeyHandler(authObject auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !requireAdminPassword(w, authObject, req.Password) {
			return
		}

		key, err := authObject.SetApiKey()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonWrite(w, map[string]string{"key": key})
	}
}

