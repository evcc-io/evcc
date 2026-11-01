package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util/locale"
	"golang.org/x/text/language"
)

// historyRange checks the database and parses the optional RFC3339 `from` and
// `to` query parameters, answering the request itself when either fails
func historyRange(w http.ResponseWriter, r *http.Request) (from, to time.Time, ok bool) {
	if db.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return from, to, false
	}
	for name, dst := range map[string]*time.Time{"from": &from, "to": &to} {
		s := r.URL.Query().Get(name)
		if s == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid '%s' parameter", name))
			return from, to, false
		}
		*dst = t
	}
	return from, to, true
}

// cacheUntilNextSlot marks the response fresh until the next slot boundary, when
// the data may change
func cacheUntilNextSlot(w http.ResponseWriter) {
	maxAge := time.Until(time.Now().Truncate(tariff.SlotDuration).Add(tariff.SlotDuration))
	w.Header().Set("Cache-Control", fmt.Sprintf("private, max-age=%d", int(maxAge.Seconds())))
}

// energyHistoryHandler returns aggregated energy history data
func energyHistoryHandler(w http.ResponseWriter, r *http.Request) {
	from, to, ok := historyRange(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()

	aggregate := q.Get("aggregate")
	if aggregate == "" {
		aggregate = "15m"
	}

	grouped := q.Get("grouped") == "true"

	filter := metrics.EnergyFilter{
		Group: q.Get("group"),
		Name:  q.Get("name"),
		Title: q.Get("title"),
	}

	res, err := metrics.QueryEnergy(from, to, aggregate, grouped, filter)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	format := q.Get("format")

	if format == "json" {
		jsonAttachment(w, res, historyFilename(from, aggregate))
		return
	}

	if format == "csv" || format == "xlsx" {
		lang := q.Get("lang")
		if lang == "" {
			if tags, _, err := language.ParseAcceptLanguage(r.Header.Get("Accept-Language")); err == nil && len(tags) > 0 {
				lang = tags[0].String()
			}
		}
		ctx := context.WithValue(context.Background(), locale.Locale, lang)
		exportResult(ctx, w, format, metrics.SeriesExport(res), historyFilename(from, aggregate))
		return
	}

	cacheUntilNextSlot(w)

	jsonWrite(w, res)
}

// energyFlowHandler returns the source to sink energy attribution of a period
func energyFlowHandler(w http.ResponseWriter, r *http.Request) {
	from, to, ok := historyRange(w, r)
	if !ok {
		return
	}
	res, err := metrics.QueryFlow(from, to)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	cacheUntilNextSlot(w)

	jsonWrite(w, res)
}

// tariffHistoryHandler returns the persisted tariff slots in [from,to)
func tariffHistoryHandler(w http.ResponseWriter, r *http.Request) {
	from, to, ok := historyRange(w, r)
	if !ok {
		return
	}
	res, err := metrics.QueryTariffs(from, to, r.URL.Query().Get("aggregate"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	cacheUntilNextSlot(w)

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
