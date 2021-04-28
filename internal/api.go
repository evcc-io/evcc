package internal

import "github.com/gorilla/mux"

type WebController interface {
	WebControl(*mux.Router)
}
