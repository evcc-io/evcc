package tibber

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /vehicles", getVehicles)

	service.Register("tibber", mux)
}

// getVehicles lists the external ids of the vehicles in the account, driving
// vehicle selection in the template. It reuses the OAuth instance created for
// the same client, so results appear once the user has authorized via the UI.
func getVehicles(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := req.URL.Query()
	clientID, clientSecret, redirectURI := q.Get("clientid"), q.Get("clientsecret"), q.Get("redirecturi")

	ids := []string{}
	defer func() { _ = json.NewEncoder(w).Encode(ids) }()

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

	for _, v := range vehicles {
		ids = append(ids, v.VIN())
	}
	slices.Sort(ids)
}
