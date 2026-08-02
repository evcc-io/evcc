package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/assistant"
	"github.com/evcc-io/evcc/server/db/settings"
)

// assistantModelsHandler lists the models the given endpoint offers. The configuration
// is posted since it is answered while it is still being edited.
func assistantModelsHandler(w http.ResponseWriter, r *http.Request) {
	var cfg assistant.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// the form sends the masked placeholder for an unchanged token
	if cfg.Token == masked {
		var stored assistant.Config
		if err := settings.Json(keys.Assistant, &stored); err == nil {
			cfg.Token = stored.Token
		}
	}

	models, err := assistant.Models(r.Context(), cfg)
	if err != nil {
		jsonError(w, http.StatusBadGateway, err)
		return
	}

	jsonWrite(w, models)
}
