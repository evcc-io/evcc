package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/db"
	dbserver "github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/locale"
	"github.com/gorilla/mux"
	"golang.org/x/text/language"
)

func csvResult(ctx context.Context, w http.ResponseWriter, res any) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="sessions.csv"`)

	if ww, ok := res.(api.CsvWriter); ok {
		_ = ww.WriteCsv(ctx, w)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// sessionHandler returns the list of charging sessions
func sessionHandler(w http.ResponseWriter, r *http.Request) {
	if dbserver.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	var res db.Sessions

	if txn := dbserver.Instance.Table("sessions").Where("charged_kwh>=0.05").Order("created desc").Find(&res); txn.Error != nil {
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
		csvResult(ctx, w, &res)
		return
	}

	jsonResult(w, res)
}

// deleteSessionHandler removes session in sessions table with given id
func deleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	if dbserver.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	var res db.Sessions

	vars := mux.Vars(r)
	id := vars["id"]

	if txn := dbserver.Instance.Table("sessions").Delete(&res, id); txn.Error != nil {
		jsonError(w, http.StatusBadRequest, txn.Error)
		return
	}

	jsonResult(w, res)
}

// updateSessionHandler updates the data of an existing session
func updateSessionHandler(w http.ResponseWriter, r *http.Request) {
	if dbserver.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var session map[string]any
	if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	if txn := dbserver.Instance.Table("sessions").Where("id = ?", id).Updates(&session); txn.Error != nil {
		jsonError(w, http.StatusBadRequest, txn.Error)
		return
	}
}
