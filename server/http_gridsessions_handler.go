package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/hems/smartgrid"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/locale"
	"golang.org/x/text/language"
)

// gridSessionsHandler returns the list of grid sessions
func gridSessionsHandler(w http.ResponseWriter, r *http.Request) {
	if db.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	var res smartgrid.GridSessions

	if txn := db.Instance.Order("created DESC").Find(&res); txn.Error != nil {
		jsonError(w, http.StatusInternalServerError, txn.Error)
		return
	}

	if r.URL.Query().Get("format") == "csv" {
		lang := r.URL.Query().Get("lang")
		if lang == "" {
			// get request language
			lang = r.Header.Get("Accept-Language")
			if tags, _, err := language.ParseAcceptLanguage(lang); err == nil && len(tags) > 0 {
				lang = tags[0].String()
			}
		}

		ctx := context.WithValue(context.Background(), locale.Locale, lang)
		csvResult(ctx, w, &res, "gridsessions")
		return
	}

	jsonWrite(w, res)
}
