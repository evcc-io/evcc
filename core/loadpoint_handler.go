package core

import (
	"encoding/json"
	"net/http"

	"github.com/andig/evcc/server/config"
	"github.com/fatih/structs"
)

func (lp *LoadPoint) configHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		lp.getConfig(w, r)
	case http.MethodPost:
		lp.setConfig(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (lp *LoadPoint) getConfig(w http.ResponseWriter, r *http.Request) {
	meta := config.Annotate(lp.LoadPointConfig)

	if err := json.NewEncoder(w).Encode(meta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (lp *LoadPoint) setConfig(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&lp.LoadPointConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lp.log.FATAL.Println(structs.Map(lp.LoadPointConfig))
}
