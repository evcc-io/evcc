package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/messenger"
	"github.com/gorilla/handlers"
)

// RegisterAppPushHandlers adds the companion app push token endpoints
func (s *HTTPd) RegisterAppPushHandlers(m *messenger.AppPush) {
	api := s.Router().PathPrefix("/api").Subrouter()
	api.Use(jsonHandler)
	api.Use(handlers.CompressHandler)
	api.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type"}),
	))

	routes := map[string]route{
		"registerpushtoken":   {"POST", "/push/token", pushTokenHandler(m.Register)},
		"unregisterpushtoken": {"DELETE", "/push/token", pushTokenHandler(m.Unregister)},
	}

	for _, r := range routes {
		api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
	}
}

func pushTokenHandler(fun func(string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token string `json:"token"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if !messenger.ValidPushToken(req.Token) {
			jsonError(w, http.StatusBadRequest, errors.New("invalid push token"))
			return
		}

		fun(req.Token)
		jsonWrite(w, true)
	}
}
