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
		w.WriteHeader(http.StatusCreated)
	}
}

// read jwt from header and cookie
func jwtFromRequest(r *http.Request) string {
	// read from header
	authHeader := r.Header.Get("Authorization")
	if token, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		return token
	}

	// read from cookie
	if cookie, _ := r.Cookie(authCookieName); cookie != nil {
		return cookie.Value
	}

	return ""
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
		ok, err := authObject.ValidateJwtToken(jwtFromRequest(r))
		if err != nil || !ok {
			w.Write([]byte("false"))
			return
		}
		w.Write([]byte("true"))
	}
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

		lifetime := time.Hour * 24 * 90 // 90 day valid
		tokenString, err := authObject.GenerateJwtToken(lifetime)
		if err != nil {
			http.Error(w, "Failed to generate JWT token.", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     authCookieName,
			Value:    tokenString,
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Now().Add(lifetime),
			SameSite: http.SameSiteStrictMode,
		})
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
			if authObject.GetAuthMode() == auth.Disabled {
				next.ServeHTTP(w, r)
				return
			}

			if authObject.GetAuthMode() == auth.Locked {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// check jwt token
			ok, err := authObject.ValidateJwtToken(jwtFromRequest(r))
			if !ok || err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// all clear, continue
			next.ServeHTTP(w, r)
		})
	}
}
