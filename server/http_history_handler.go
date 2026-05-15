package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/locale"
	"golang.org/x/text/language"
)

// energyHistoryHandler returns aggregated energy history data
func energyHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if db.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	q := r.URL.Query()

	var from, to time.Time

	if s := q.Get("from"); s != "" {
		var err error
		if from, err = time.Parse(time.RFC3339, s); err != nil {
			jsonError(w, http.StatusBadRequest, errors.New("invalid 'from' parameter"))
			return
		}
	}

	if s := q.Get("to"); s != "" {
		var err error
		if to, err = time.Parse(time.RFC3339, s); err != nil {
			jsonError(w, http.StatusBadRequest, errors.New("invalid 'to' parameter"))
			return
		}
	}

	aggregate := q.Get("aggregate")
	if aggregate == "" {
		aggregate = "15m"
	}

	grouped := q.Get("grouped") == "true"

	res, err := metrics.QueryEnergy(from, to, aggregate, grouped)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	if q.Get("format") == "csv" {
		lang := q.Get("lang")
		if lang == "" {
			if tags, _, err := language.ParseAcceptLanguage(r.Header.Get("Accept-Language")); err == nil && len(tags) > 0 {
				lang = tags[0].String()
			}
		}
		ctx := context.WithValue(context.Background(), locale.Locale, lang)
		csvResult(ctx, w, metrics.SeriesCSV(res), historyFilename(from, aggregate))
		return
	}

	jsonWrite(w, res)
}

// historyFilename returns history-energy-YYYY-MM-DD / -YYYY-MM / -YYYY
// for day/month/year aggregates.
func historyFilename(from time.Time, aggregate string) string {
	if from.IsZero() {
		return "history-energy"
	}
	format := "2006-01-02"
	switch aggregate {
	case "day":
		format = "2006-01"
	case "month":
		format = "2006"
	}
	return "history-energy-" + from.Local().Format(format)
}
