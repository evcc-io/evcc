package tibber

import (
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /vehicles", getVehicles)

	service.Register("tibber", mux)
}

// vehicleOptions maps the device list to suggestions keyed by VIN
func vehicleOptions(vehicles []Device) []service.Option {
	res := make([]service.Option, 0, len(vehicles))
	for _, v := range vehicles {
		model := strings.TrimSpace(v.Info.Brand + " " + v.Info.Model)
		// every vehicle is a valid choice, a single one auto-fills the vin
		o := service.Option{Value: v.VIN(), Label: strings.TrimSpace(v.Info.Name), Match: true}
		if o.Label == "" {
			o.Label = model
		} else {
			o.Hint = model
		}
		res = append(res, o)
	}
	slices.SortFunc(res, func(a, b service.Option) int { return cmp.Compare(a.Value, b.Value) })
	return res
}

// getVehicles lists the vehicles in the account, driving vehicle selection in
// the template. It reuses the OAuth instance created for the same client, so
// results appear once the user has authorized via the UI.
func getVehicles(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := req.URL.Query()
	clientID, clientSecret, redirectURI := q.Get("clientid"), q.Get("clientsecret"), q.Get("redirecturi")

	res := []service.Option{}
	defer func() { _ = json.NewEncoder(w).Encode(res) }()

	if clientID == "" || clientSecret == "" {
		return
	}

	log := util.NewLogger("tibber").Redact(clientID, clientSecret)

	ts, err := NewOAuth(util.WithLogger(context.Background(), log), clientID, clientSecret, redirectURI, "")
	if err != nil {
		return
	}

	// no values until the user has authorized
	vehicles, err := NewAPI(log, ts).Vehicles()
	if err != nil {
		return
	}

	res = vehicleOptions(vehicles)
}
