package eebus

import (
	"cmp"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	shipapi "github.com/enbility/ship-go/api"
	"github.com/evcc-io/evcc/server/service"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /services", getServices)
	mux.HandleFunc("GET /pairings", getPairings)
	mux.HandleFunc("DELETE /pairings/{id}", deletePairing)

	service.Register("eebus", mux)
}

// deviceOptions maps discovered services to suggestions, used marks skis already configured
func deviceOptions(services []shipapi.RemoteMdnsService, used func(ski string) bool) []service.Option {
	res := make([]service.Option, 0, len(services))
	for _, s := range services {
		res = append(res, service.Option{
			Value: s.Ski,
			Label: cmp.Or(strings.TrimSpace(s.Brand+" "+s.Model), s.Name, s.ShipID),
			Hint:  cmp.Or(s.Serial, s.ShipID),
			Match: true, // match keeps the single-device auto-fill
			Used:  used(s.Ski),
		})
	}
	slices.SortFunc(res, func(a, b service.Option) int { return cmp.Compare(a.Value, b.Value) })
	return res
}

func getServices(w http.ResponseWriter, req *http.Request) {
	res := []service.Option{}
	if instance != nil {
		res = deviceOptions(instance.RemoteServices(), instance.HasDevices)
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
