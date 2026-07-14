package octopuskraken

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
)

func init() {
	register("octopus-de", BaseURI)
	register("octopus-it", ItBaseURI)
}

// register wires the device-listing endpoint for a Kraken instance under the given service name.
func register(name, baseURI string) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /devices", func(w http.ResponseWriter, req *http.Request) {
		getDevices(w, req, name, baseURI)
	})

	service.Register(name, mux)
}

// getDevices lists the ids of the SmartFlex devices in the account, driving
// device selection in the template.
func getDevices(w http.ResponseWriter, req *http.Request, name, baseURI string) {
	w.Header().Set("Content-Type", "application/json")

	q := req.URL.Query()
	email, password, account := q.Get("email"), q.Get("password"), q.Get("accountnumber")

	ids := []string{}
	defer func() { _ = json.NewEncoder(w).Encode(ids) }()

	if email == "" || password == "" {
		return
	}

	log := util.NewLogger(name).Redact(email, password)
	api, err := NewAPI(log, baseURI, email, password)
	if err != nil {
		log.ERROR.Println(err)
		return
	}

	account, err = api.Account(account)
	if err != nil {
		log.ERROR.Println(err)
		return
	}

	devices, err := api.Devices(account)
	if err != nil {
		log.ERROR.Println(err)
		return
	}

	for _, d := range devices {
		ids = append(ids, d.ID)
	}
	slices.Sort(ids)
}
