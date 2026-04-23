package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/server/db"
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

	res, err := metrics.QueryImportEnergy(from, to, aggregate, grouped)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	jsonWrite(w, res)
}
