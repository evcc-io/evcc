package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/assistant"
	"github.com/evcc-io/evcc/server/db/settings"
)

// assistantToken restores the stored token for the endpoint it was stored for. A
// request naming another url has to bring its own, the stored one must not leak there.
func assistantToken(req, stored assistant.Config) string {
	if req.Token != masked {
		return req.Token
	}

	if req.Provider == stored.Provider && req.BaseUrl == stored.BaseUrl {
		return stored.Token
	}

	return ""
}

// assistantProvidersHandler describes the supported providers to the config ui
func assistantProvidersHandler(w http.ResponseWriter, r *http.Request) {
	jsonWrite(w, assistant.Providers())
}

// assistantModelsHandler lists the models the given endpoint offers. The configuration
// is posted since it is answered while it is still being edited.
func assistantModelsHandler(w http.ResponseWriter, r *http.Request) {
	var cfg assistant.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// the form sends the masked placeholder for an unchanged token
	var stored assistant.Config
	_ = settings.Json(keys.Assistant, &stored)
	cfg.Token = assistantToken(cfg, stored)

	models, err := assistant.Models(r.Context(), cfg)
	if err != nil {
		jsonError(w, http.StatusBadGateway, err)
		return
	}

	jsonWrite(w, models)
}
