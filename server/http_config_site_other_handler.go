package server

import (
	"net/http"
	"time"
)

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

// // maxgridsupplywhilebatterycharging
// func maxGridSupplyWhileBatteryChargingHandler(w http.ResponseWriter, r *http.Request) {
// 	jsonResult(w, 42)
// }

// func updateMaxGridSupplyWhileBatteryChargingHandler(w http.ResponseWriter, r *http.Request) {
// 	w.WriteHeader(http.StatusOK)
// }
