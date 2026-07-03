package eebus

import (
	"encoding/json"
	"net/http"

	shipapi "github.com/enbility/ship-go/api"
	"github.com/evcc-io/evcc/server/service"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /services", getServices)
	mux.HandleFunc("GET /pairings", getPairings)
	mux.HandleFunc("DELETE /pairings", deletePairings)

	service.Register("eebus", mux)
}

func getServices(w http.ResponseWriter, req *http.Request) {
	var res []string
	if instance != nil {
		for _, s := range instance.RemoteServices() {
			res = append(res, s.Ski)
		}
	}
	json.NewEncoder(w).Encode(res)
}

// getPairings returns the devices paired via the SHIP Pairing Service
func getPairings(w http.ResponseWriter, req *http.Request) {
	res := []shipapi.ServiceIdentity{}
	if instance != nil {
		if paired := instance.Paired(); paired != nil {
			res = append(res, *paired)
		}
	}
	json.NewEncoder(w).Encode(res)
}

// deletePairings removes the SHIP Pairing Service pairing and revokes trust
func deletePairings(w http.ResponseWriter, req *http.Request) {
	if instance != nil {
		instance.RemovePairing()
	}
	w.WriteHeader(http.StatusOK)
}
