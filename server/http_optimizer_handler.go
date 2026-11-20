package server

import (
	"cmp"
	"net/http"

	"github.com/evcc-io/evcc/core"
)

// optimizerExplainHandler explains the last optimizer plan in the language of the lang
// query parameter, or of the Accept-Language header without one
func optimizerExplainHandler(site *core.Site) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := cmp.Or(r.URL.Query().Get("lang"), getPreferredLanguage(r.Header.Get("Accept-Language")))

		if err := site.ExplainOptimizerResult(r.Context(), lang); err != nil {
			// relayed from the optimizer service, the ui shows the message in place
			jsonError(w, http.StatusBadGateway, err)
			return
		}

		// the explanation is published as evopt-explanation
		w.WriteHeader(http.StatusNoContent)
	}
}

// optimizerExplainFeedbackHandler relays a vote on the last explanation to the optimizer service
func optimizerExplainFeedbackHandler(site *core.Site) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Vote    string `json:"vote"`
			Comment string `json:"comment"`
		}

		if err := jsonDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := site.ExplainFeedback(r.Context(), req.Vote, req.Comment); err != nil {
			jsonError(w, http.StatusBadGateway, err)
			return
		}

		jsonWrite(w, struct{}{})
	}
}
