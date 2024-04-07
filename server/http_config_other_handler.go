package server

import (
	"encoding/json"
	"net/http"
	"time"
)

type mqttPayload struct {
	Broker   string `json:"broker"`
	Topic    string `json:"topic"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func mqttHandler(w http.ResponseWriter, r *http.Request) {
	res := mqttPayload{
		Broker:   "localhost:1883",
		Topic:    "evcc",
		User:     "",
		Password: "",
	}

	jsonResult(w, res)
}

func updateMqttHandler(w http.ResponseWriter, r *http.Request) {
	var payload mqttPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type influxPayload struct {
	Url      string `json:"url"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func influxHandler(w http.ResponseWriter, r *http.Request) {
	res := influxPayload{
		Url:      "http://localhost:8086",
		Database: "evcc",
		User:     "",
		Password: "",
	}

	jsonResult(w, res)
}

func updateInfluxHandler(w http.ResponseWriter, r *http.Request) {
	var payload influxPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type eebusPayload struct {
	Yaml string `json:"yaml"`
}

func eebusHandler(w http.ResponseWriter, r *http.Request) {
	res := eebusPayload{
		Yaml: "",
	}

	jsonResult(w, res)
}

func updateEebusHandler(w http.ResponseWriter, r *http.Request) {
	var payload eebusPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type tariffsPayload struct {
	Currency   string `json:"currency"`
	GridYaml   string `json:"gridYaml"`
	FeedinYaml string `json:"feedinYaml"`
	Co2Yaml    string `json:"co2Yaml"`
}

func tariffsHandler(w http.ResponseWriter, r *http.Request) {
	res := tariffsPayload{
		Currency:   "EUR",
		GridYaml:   "",
		FeedinYaml: "",
		Co2Yaml:    "",
	}

	jsonResult(w, res)
}

func updateTariffsHandler(w http.ResponseWriter, r *http.Request) {
	var payload tariffsPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type messagingPayload struct {
	EventsYaml   string `json:"eventsYaml"`
	ServicesYaml string `json:"servicesYaml"`
}

func messagingHandler(w http.ResponseWriter, r *http.Request) {
	res := messagingPayload{
		EventsYaml:   "",
		ServicesYaml: "",
	}

	jsonResult(w, res)
}

func updateMessagingHandler(w http.ResponseWriter, r *http.Request) {
	var payload messagingPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type modbusProxyPayload struct {
	Yaml string `json:"yaml"`
}

func modbusProxyHandler(w http.ResponseWriter, r *http.Request) {
	res := modbusProxyPayload{
		Yaml: "",
	}

	jsonResult(w, res)
}

func updateModbusProxyHandler(w http.ResponseWriter, r *http.Request) {
	var payload modbusProxyPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func sponsorStatusHandler(w http.ResponseWriter, r *http.Request) {
	// @andig we should not return the sponsor token here, instead I modeled the sponsorship status.
	// But maybe it's also a good idea to not implement this single get endpoint at all.
	// We could build a  separate endpoint returning the state of all configuration.
	// Something like /api/config/state which is auth-only and returns status info to all config entities (device dump data, mqtt configured? connected?, eebus configured?, sponsorship active?, ...)

	res := struct {
		Valid   bool      `json:"valid"`
		Name    string    `json:"name"`
		Demo    bool      `json:"demo"`
		Expires time.Time `json:"expires"`
	}{
		Valid:   true,
		Name:    "sponsor@evcc.io",
		Demo:    false,
		Expires: time.Now().AddDate(1, 0, 0),
	}

	jsonResult(w, res)
}

func updateSponsortokenHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func intervalHandler(w http.ResponseWriter, r *http.Request) {
	res := struct {
		Seconds int `json:"interval"`
	}{
		Seconds: 30,
	}

	jsonResult(w, res)
}

func updateIntervalHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type networkPayload struct {
	Schema string `json:"schema"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
}

func networkHandler(w http.ResponseWriter, r *http.Request) {
	res := networkPayload{
		Schema: "http",
		Host:   "evcc.local",
		Port:   7070,
	}

	jsonResult(w, res)
}

func updateNetworkHandler(w http.ResponseWriter, r *http.Request) {
	var payload networkPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// maxgridsupplywhilebatterycharging
func maxGridSupplyWhileBatteryChargingHandler(w http.ResponseWriter, r *http.Request) {
	jsonResult(w, 42)
}

func updateMaxGridSupplyWhileBatteryChargingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
