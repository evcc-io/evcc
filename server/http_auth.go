package server

import (
	"net/http"
	"time"

	"github.com/evcc-io/evcc/core/auth"
)

var authCookieName = "auth"

func setPasswordHandler(auth auth.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := auth.SetAdminPassword(r.FormValue("password")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func loginHandler(auth auth.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		password := r.FormValue("password")
		if !auth.IsAdminPasswordValid(password) {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		lifetime := time.Hour * 24 // 1 day valid
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
		Expires:  time.Now().Add(-1),
	})
}

func ensureAuth(auth auth.API, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// read auth cookie
		cookie, err := r.Cookie(authCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// check jwt token
		ok, err := auth.ValidateJwtToken(cookie.Value)
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
