package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/locale"
	"github.com/gorilla/mux"
	"golang.org/x/text/language"
)

func csvResult(ctx context.Context, w http.ResponseWriter, res any, filename string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.csv"`)

	if ww, ok := res.(api.CsvWriter); ok {
		_ = ww.WriteCsv(ctx, w)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// sessionHandler returns the list of charging sessions
func sessionHandler(w http.ResponseWriter, r *http.Request) {
	if db.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	var res session.Sessions

	filename := "session"
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	query := db.Instance.Where("charged_kwh >= ?", 0.05).Order("created DESC")

	if year > 0 && month > 0 {
		filename += fmt.Sprintf("-%04d-%02d", year, month)
		l, _ := time.LoadLocation("Local")
		first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, l)
		last := first.AddDate(0, 1, 0).Add(-1)
		query = query.Where("created BETWEEN ? AND ?", first, last)
	} else if year > 0 {
		filename += fmt.Sprintf("%04d", year, month)
		l, _ := time.LoadLocation("Local")
		first := time.Date(year, time.January, 1, 0, 0, 0, 0, l)
		last := first.AddDate(1, 0, 0).Add(-1)
		query = query.Where("created BETWEEN ? AND ?", first, last)
	}

	if txn := query.Find(&res); txn.Error != nil {
		jsonError(w, http.StatusInternalServerError, txn.Error)
		return
	}

	// prepare data
	for i, s := range res {
		if s.Odometer != nil {
			odo := math.Round(*s.Odometer*10) / 10
			res[i].Odometer = &odo
		}
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
		csvResult(ctx, w, &res, filename)
		return
	}

	jsonResult(w, res)
}

// deleteSessionHandler removes session in sessions table with given id
func deleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	if db.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	var res session.Sessions

	vars := mux.Vars(r)
	id := vars["id"]

	if txn := db.Instance.Table("sessions").Delete(&res, id); txn.Error != nil {
		jsonError(w, http.StatusBadRequest, txn.Error)
		return
	}

	jsonResult(w, res)
}

// updateSessionHandler updates the data of an existing session
func updateSessionHandler(w http.ResponseWriter, r *http.Request) {
	if db.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	id := mux.Vars(r)["id"]

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		jsonError(w, http.StatusBadRequest, errors.New("invalid JSON"))
		return
	}

	updates := map[string]interface{}{}
	for _, field := range []string{"vehicle", "loadpoint"} {
		if val, ok := data[field]; ok {
			updates[field] = val
		}
	}

	if len(updates) == 0 {
		jsonError(w, http.StatusBadRequest, errors.New("no valid fields to update"))
		return
	}

	// https://github.com/evcc-io/evcc/issues/13738#issuecomment-2094070362
	if txn := db.Instance.Table("sessions").Where("id = ?", id).Updates(updates); txn.Error != nil {
		jsonError(w, http.StatusBadRequest, txn.Error)
		return
	}
}
