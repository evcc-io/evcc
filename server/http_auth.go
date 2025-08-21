package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/auth"
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

// read token from header and cookie
func tokenFromRequest(r *http.Request) string {
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

// authStatusHandler login status (true/false) based on token. Errors if admin password is not configured
func authStatusHandler(authObject auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authObject.GetAuthMode() == auth.Disabled {
			fmt.Fprint(w, "true")
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
		_, err := authObject.ValidateToken(tokenFromRequest(r))
		fmt.Fprint(w, strconv.FormatBool(err == nil))
	}
}

func loginRequired(authObject auth.Auth, r *http.Request) error {
	if authObject.GetAuthMode() == auth.Locked {
		return errors.New("forbidden in demo mode")
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	if !authObject.IsAdminPasswordValid(req.Password) {
		return errors.New("invalid password")
	}

	return nil
}

func loginHandler(authObject auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := loginRequired(authObject, r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		lifetime := 90 * 24 * time.Hour // 90 day valid
		token, err := authObject.GenerateToken(auth.JwtToken, lifetime)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     authCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Now().Add(lifetime),
			SameSite: http.SameSiteStrictMode,
		})
	}
}

func tokenHandler(authObject auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := loginRequired(authObject, r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		lifetime := 365 * 24 * time.Hour
		token, err := authObject.GenerateToken(auth.ApiToken, lifetime)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		jsonWrite(w, token)
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

			// check token
			typ, err := authObject.ValidateToken(tokenFromRequest(r))
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// all clear, continue
			ctx := context.WithValue(r.Context(), auth.ContextAuthType, typ)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
