package service

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/server/service"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /single", getSingle)
	mux.HandleFunc("GET /country", getCountry)
	mux.HandleFunc("GET /{country}/city", getCity)
	mux.HandleFunc("GET /modbus", getModbus)

	service.Register("demo", mux)
}

func getSingle(w http.ResponseWriter, req *http.Request) {
	json.NewEncoder(w).Encode([]string{"demo-value"})
}

func getCountry(w http.ResponseWriter, req *http.Request) {
	json.NewEncoder(w).Encode([]string{"germany", "france", "spain"})
}

func getCity(w http.ResponseWriter, req *http.Request) {
	country := req.PathValue("country")
	var cities []string
	switch country {
	case "germany":
		cities = []string{"berlin", "munich", "hamburg"}
	case "france":
		cities = []string{"paris", "lyon", "marseille"}
	case "spain":
		cities = []string{"madrid", "barcelona", "valencia"}
	}
	json.NewEncoder(w).Encode(cities)
}

func getModbus(w http.ResponseWriter, req *http.Request) {
	// Verify that either uri or device is provided (mimics modbus connection params)
	uri := req.URL.Query().Get("uri")
	device := req.URL.Query().Get("device")
	address := req.URL.Query().Get("address")
	id := req.URL.Query().Get("id")

	if uri == "" && device == "" {
		http.Error(w, "either uri or device parameter required", http.StatusBadRequest)
		return
	}

	if address == "" {
		http.Error(w, "address parameter required", http.StatusBadRequest)
		return
	}

	// Return different values based on connection type and id
	// Format: address,id:id,type (e.g., "100,id:2,tcp")
	connType := "tcp"
	if device != "" {
		connType = "serial"
	}
	result := address + ",id:" + id + "," + connType
	json.NewEncoder(w).Encode([]string{result})
}
