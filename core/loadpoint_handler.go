package core

import (
	"encoding/json"
	"net/http"

	"github.com/fatih/structs"
)

func (s *LoadPoint) configHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getConfig(w, r)
	case http.MethodPost:
		s.setConfig(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusBadRequest)
	}
}

func (s *LoadPoint) getConfig(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(s.LoadPointConfig); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *LoadPoint) setConfig(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&s.LoadPointConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.log.FATAL.Println(structs.Map(s.LoadPointConfig))
}
