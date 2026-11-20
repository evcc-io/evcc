package octopuskraken

import (
	"cmp"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

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

// deviceTypes maps the Kraken device type enum to a readable hint.
var deviceTypes = map[string]string{
	"ELECTRIC_VEHICLES": "Vehicle",
	"CHARGE_POINTS":     "Charge point",
}

func deviceOptions(devices []Device) []service.Option {
	res := make([]service.Option, 0, len(devices))
	for _, d := range devices {
		parts := []string{cmp.Or(deviceTypes[d.DeviceType], d.DeviceType), d.Provider}
		res = append(res, service.Option{
			Value: d.ID,
			Label: d.Name,
			Hint:  strings.Join(slices.DeleteFunc(parts, func(s string) bool { return s == "" }), " · "),
			// vehicle template: charge points are listed but don't match
			Match: d.DeviceType == "ELECTRIC_VEHICLES",
		})
	}
	slices.SortFunc(res, func(a, b service.Option) int { return cmp.Compare(a.Value, b.Value) })
	return res
}

// getDevices lists the SmartFlex devices in the account, driving
// device selection in the template.
func getDevices(w http.ResponseWriter, req *http.Request, name, baseURI string) {
	w.Header().Set("Content-Type", "application/json")

	q := req.URL.Query()
	email, password, account := q.Get("email"), q.Get("password"), q.Get("accountnumber")

	res := []service.Option{}
	defer func() { _ = json.NewEncoder(w).Encode(res) }()

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

	res = deviceOptions(devices)
}
