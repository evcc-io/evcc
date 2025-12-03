package service

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/server/service"
	serialports "go.bug.st/serial"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", getSerialPorts)

	service.Register("serial", mux)
}

func getSerialPorts(w http.ResponseWriter, req *http.Request) {
	ports, err := serialports.GetPortsList()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ports)
}
