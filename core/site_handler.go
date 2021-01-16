package core

import (
	"encoding/json"
	"net/http"

	"github.com/fatih/structs"
	"github.com/gorilla/mux"
)

// RegisterConfigHandler registers the configuration handler with the HTTP server
func (s *Site) RegisterConfigHandler(router *mux.Router) {
	router.PathPrefix("/site").HandlerFunc(s.configHandler)
}

func (s *Site) configHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getConfig(w, r)
	case http.MethodPost:
		s.setConfig(w, r)
	default:
		http.Error(w, "", http.StatusBadRequest)
	}
}

func (s *Site) getConfig(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(s.SiteConfig); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Site) setConfig(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&s.SiteConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.log.FATAL.Println(structs.Map(s.SiteConfig))
}
