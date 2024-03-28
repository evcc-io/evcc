package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util/auth"
)

const authCookieName = "auth"

type updatePasswordRequest struct {
	Current string `json:"current"`
	New     string `json:"new"`
}

type loginRequest struct {
	Password string `json:"password"`
}

func updatePasswordHandler(auth auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updatePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// update password
		if auth.IsAdminPasswordConfigured() {
			if !auth.IsAdminPasswordValid(req.Current) {
				http.Error(w, "Invalid password", http.StatusBadRequest)
				return
			}

			if err := auth.SetAdminPassword(req.New); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusAccepted)
			return
		}

		// create new password
		if err := auth.SetAdminPassword(req.New); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

// read jwt from header and cookie
func jwtFromRequest(r *http.Request) string {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		if cookie, _ := r.Cookie(authCookieName); cookie != nil {
			tokenString = cookie.Value
		}
	}

	return tokenString
}

// authStatusHandler login status (true/false) based on jwt token. Error if admin password is not configured
func authStatusHandler(auth auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsAdminPasswordConfigured() {
			http.Error(w, "Not implemented", http.StatusNotImplemented)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		ok, err := auth.ValidateJwtToken(jwtFromRequest(r))
		if err != nil || !ok {
			w.Write([]byte("false"))
			return
		}
		w.Write([]byte("true"))
	}
}

func loginHandler(auth auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !auth.IsAdminPasswordValid(req.Password) {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		lifetime := time.Hour * 24 * 90 // 90 day valid
		tokenString, err := auth.GenerateJwtToken(lifetime)
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

func ensureAuth(auth auth.Auth, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// check jwt token
		ok, err := auth.ValidateJwtToken(jwtFromRequest(r))
		if !ok || err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// all clear, continue
		next.ServeHTTP(w, r)
	}
}