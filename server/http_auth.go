package server

import (
	"io"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/core/site"
)

var authCookieName = "auth"
var authQueryParam = "auth"

func setPasswordHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		password, _ := io.ReadAll(r.Body)
		if err := site.Auth().SetAdminPassword(string(password)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// read jwt from 1) Authorization header, 2) cookie, 3) query parameter
func jwtFromRequest(r *http.Request) string {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		cookie, _ := r.Cookie(authCookieName)
		if cookie != nil {
			tokenString = cookie.Value
		}
	}
	if tokenString == "" {
		tokenString = r.URL.Query().Get(authQueryParam)
	}

	return tokenString
}

// authStatusHandler login status (true/false) based on jwt token. Error if admin password is not configured
func authStatusHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := site.Auth()
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

func loginHandler(site site.API) http.HandlerFunc {
	auth := site.Auth()
	return func(w http.ResponseWriter, r *http.Request) {
		password := r.FormValue("password")
		if !auth.IsAdminPasswordValid(password) {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		lifetime := time.Hour * 24 * 90 // 90 day valid
		tokenString, err := auth.GenerateJwtToken(lifetime)
		if err != nil {
			http.Error(w, "Failed to generate JWT token: "+err.Error(), http.StatusInternalServerError)
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
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   0,
	})
}

func ensureAuth(site site.API, next http.HandlerFunc) http.HandlerFunc {
	auth := site.Auth()
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

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world!"))
}
