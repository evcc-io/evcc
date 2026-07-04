package eebus

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/server/service"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /services", getServices)
	mux.HandleFunc("GET /pairings", getPairings)
	mux.HandleFunc("DELETE /pairings/{id}", deletePairing)

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

// getPairings returns all trusted devices, tagged by how trust was established
func getPairings(w http.ResponseWriter, req *http.Request) {
	res := []PairingInfo{}
	if instance != nil {
		res = append(res, instance.Pairings()...)
	}
	json.NewEncoder(w).Encode(res)
}

// deletePairing removes a single pairing identified by ship id or ski
func deletePairing(w http.ResponseWriter, req *http.Request) {
	if instance == nil || !instance.RemovePairing(req.PathValue("id")) {
		w.WriteHeader(http.StatusNotFound)
	}
}
