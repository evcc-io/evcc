package core

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andig/evcc/server/config"
	"github.com/fatih/structs"
	"github.com/gorilla/mux"
)

// RegisterConfigHandler registers the configuration handler with the HTTP server
func (s *Site) RegisterConfigHandler(router *mux.Router) {
	router.PathPrefix("/site").HandlerFunc(s.configHandler)

	for idx, lp := range s.loadpoints {
		router.PathPrefix(fmt.Sprintf("/loadpoints/%d", idx)).HandlerFunc(lp.configHandler)
	}
}

func (s *Site) configHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getConfig(w, r)
	case http.MethodPost:
		s.setConfig(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Site) getConfig(w http.ResponseWriter, r *http.Request) {
	meta := config.Annotate(s.SiteConfig)

	if err := json.NewEncoder(w).Encode(meta); err != nil {
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

	s.log.ERROR.Println("received:", structs.Map(s.SiteConfig))
}
