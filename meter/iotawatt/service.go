package iotawatt

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /series", getSeries)
	mux.HandleFunc("GET /config", getConfig)

	service.Register("iotawatt", mux)
}

// configResponse is the response from the /config service endpoint.
type configResponse struct {
	ThreePhase bool              `json:"threephase"`
	Phases     map[string]int    `json:"phases"` // input name -> phase (1, 2, 3)
}

func connectionFromRequest(req *http.Request) (*Connection, error) {
	uri := util.DefaultScheme(strings.TrimSuffix(req.URL.Query().Get("uri"), "/"), "http")
	if uri == "" {
		return nil, errMissingURI
	}

	return NewConnection(uri, 0)
}

// getSeries returns the available Watts series names from the IoTaWatt device.
func getSeries(w http.ResponseWriter, req *http.Request) {
	conn, err := connectionFromRequest(req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(util.ErrorAsJson(err))
		return
	}

	series, err := conn.ShowSeries()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(util.ErrorAsJson(err))
		return
	}

	// filter to Watts series only — these are the ones usable for power/current/energy
	unit := req.URL.Query().Get("unit")
	if unit == "" {
		unit = "Watts"
	}

	var result []string
	for _, s := range series {
		if strings.EqualFold(s.Unit, unit) {
			result = append(result, s.Name)
		}
	}

	w.Header().Set("Cache-Control", "max-age=10")
	json.NewEncoder(w).Encode(result)
}

// getConfig returns phase configuration from the IoTaWatt device.
// The UI uses this to determine single-phase vs three-phase mode and
// to know which phase each input belongs to.
func getConfig(w http.ResponseWriter, req *http.Request) {
	conn, err := connectionFromRequest(req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(util.ErrorAsJson(err))
		return
	}

	cfg, err := conn.DeviceConfig()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(util.ErrorAsJson(err))
		return
	}

	phases := make(map[string]int)
	phasesSeen := make(map[int]bool)
	for _, inp := range cfg.Inputs {
		if inp != nil && inp.Type == "CT" {
			p := inp.Phase()
			phases[inp.Name] = p
			phasesSeen[p] = true
		}
	}

	res := configResponse{
		ThreePhase: len(phasesSeen) == 3,
		Phases:     phases,
	}

	w.Header().Set("Cache-Control", "max-age=10")
	json.NewEncoder(w).Encode(res)
}
