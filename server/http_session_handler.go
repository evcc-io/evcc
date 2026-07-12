package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/export"
	"github.com/evcc-io/evcc/util/export/csv"
	"github.com/evcc-io/evcc/util/export/xlsx"
	"github.com/evcc-io/evcc/util/locale"
	"github.com/gorilla/mux"
	"golang.org/x/text/language"
)

// jsonAttachment writes res as json with download headers
func jsonAttachment(w http.ResponseWriter, res any, filename string) {
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.json"`)
	jsonWrite(w, res)
}

// exportResult writes res to w as format (csv|xlsx) with download headers.
func exportResult(ctx context.Context, w http.ResponseWriter, format string, res export.Writer, filename string) {
	if format == "xlsx" {
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.xlsx"`)
	} else {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.csv"`)
	}

	var ww export.RowWriter
	var err error
	if format == "xlsx" {
		ww, err = xlsx.New(ctx, w)
	} else {
		ww, err = csv.New(ctx, w)
	}
	if err == nil {
		err = res.Write(ww)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// sessionHandler returns the list of charging sessions
func sessionHandler(w http.ResponseWriter, r *http.Request) {
	if db.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	var (
		res  session.Sessions
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
		push("STRFTIME('%Y', created, 'localtime') LIKE ?", year)

		if month := fmt.Sprintf("%02s", r.URL.Query().Get("month")); month != "00" {
			filename += "-" + month
			push("STRFTIME('%m', created, 'localtime') LIKE ?", month)
		}
	}

	// TODO support other databases than Sqlite
	query := strings.Join(append([]string{"charged_kwh>=0.05"}, cond...), " AND ")
	if txn := db.Instance.Where(query, args...).Order("created DESC").Find(&res); txn.Error != nil {
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

	format := r.URL.Query().Get("format")

	if format == "json" {
		jsonAttachment(w, res, filename)
		return
	}

	if format == "csv" || format == "xlsx" {
		lang := r.URL.Query().Get("lang")
		if lang == "" {
			// get request language
			lang = r.Header.Get("Accept-Language")
			if tags, _, err := language.ParseAcceptLanguage(lang); err == nil && len(tags) > 0 {
				lang = tags[0].String()
			}
		}

		ctx := context.WithValue(context.Background(), locale.Locale, lang)
		exportResult(ctx, w, format, &res, filename)
		return
	}

	jsonWrite(w, res)
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

	jsonWrite(w, res)
}

// updateSessionHandler updates the data of an existing session
func updateSessionHandler(w http.ResponseWriter, r *http.Request) {
	if db.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	id := mux.Vars(r)["id"]

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// only update fields present in the request; a null value clears the column
	updates := map[string]any{}
	if v, ok := body["vehicle"]; ok {
		updates["vehicle"] = v
	}
	if v, ok := body["loadpoint"]; ok {
		updates["loadpoint"] = v
	}
	if v, ok := body["odometer"]; ok {
		updates["odometer"] = v
	}

	if len(updates) == 0 {
		jsonError(w, http.StatusBadRequest, errors.New("nothing to update"))
		return
	}

	if txn := db.Instance.Table("sessions").Where("id = ?", id).Updates(updates); txn.Error != nil {
		jsonError(w, http.StatusBadRequest, txn.Error)
		return
	}
}
