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
		res = append(res, instance.Pairings()...)
	}
	json.NewEncoder(w).Encode(res)
}

// deletePairings removes a single pairing identified by the id query parameter (ship id or ski)
func deletePairings(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	if id == "" || instance == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !instance.RemovePairing(id) {
		w.WriteHeader(http.StatusNotFound)
	}
}
