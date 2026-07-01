package eebus

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/server/service"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /services", getServices)

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
