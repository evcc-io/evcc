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
