package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util"
)

// deleteResult reports the number of affected rows
type deleteResult struct {
	Deleted int64 `json:"deleted"`
}

// timeRange parses the mandatory from/to query parameters
func timeRange(r *http.Request) (time.Time, time.Time, error) {
	q := r.URL.Query()

	from, err := time.Parse(time.RFC3339, q.Get("from"))
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid 'from' parameter")
	}

	to, err := time.Parse(time.RFC3339, q.Get("to"))
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid 'to' parameter")
	}

	return from, to, nil
}

// deleteEnergyHandler removes energy metrics. Parameters match
// /api/history/energy so the same request can be previewed before deleting.
func deleteEnergyHandler(w http.ResponseWriter, r *http.Request) {
	if db.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	from, to, err := timeRange(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	q := r.URL.Query()
	filter := metrics.EnergyFilter{
		Group: q.Get("group"),
		Name:  q.Get("name"),
		Title: q.Get("title"),
	}

	rows, err := metrics.DeleteEnergy(from, to, filter)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	// invalidate cached values (e.g. the solar scale) derived from the deleted history
	util.ResetCached()

	jsonWrite(w, deleteResult{rows})
}

// deleteTariffsHandler removes persisted tariff values
func deleteTariffsHandler(w http.ResponseWriter, r *http.Request) {
	if db.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	from, to, err := timeRange(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	rows, err := metrics.DeleteTariffs(from, to, r.URL.Query().Get("usage"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, metrics.ErrInvalidUsage) {
			status = http.StatusBadRequest
		}
		jsonError(w, status, err)
		return
	}

	jsonWrite(w, deleteResult{rows})
}
