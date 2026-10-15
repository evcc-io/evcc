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

		explanation, err := site.ExplainOptimizerResult(r.Context(), lang)
		if err != nil {
			// relayed from the optimizer service, the ui shows the message in place
			jsonError(w, http.StatusBadGateway, err)
			return
		}

		jsonWrite(w, map[string]string{"explanation": explanation})
	}
}
