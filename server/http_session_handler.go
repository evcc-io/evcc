package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/db"
	dbserver "github.com/evcc-io/evcc/server/db"
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
	if dbserver.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	var (
		res  db.Sessions
		cond []string
		args []any
	)

	push := func(field, val string) {
		cond = append(cond, field)
		args = append(args, val)
	}

	filename := "session"
	if year := r.URL.Query().Get("year"); year != "" {
		filename += "-" + year
		push("STRFTIME('%Y', created) LIKE ?", year)

		if month := fmt.Sprintf("%02s", r.URL.Query().Get("month")); month != "00" {
			filename += "-" + month
			push("STRFTIME('%m', created) LIKE ?", month)
		}
	}

	// TODO support other databases than Sqlite
	query := strings.Join(append([]string{"charged_kwh>=0.05"}, cond...), " AND ")
	if txn := dbserver.Instance.Where(query, args...).Order("created DESC").Find(&res); txn.Error != nil {
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
