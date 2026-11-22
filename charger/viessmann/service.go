package viessmann

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
)

var serviceMux = http.NewServeMux()

func init() {
	serviceMux.HandleFunc("GET /equipment", getEquipment)
	serviceMux.HandleFunc("GET /measurements", getMeasurements)

	service.Register("viessmann", serviceMux)
}

// errUnauthorized indicates that the user has not (yet) authorized. The config
// UI polls the service while the form is being filled, hence this is expected.
var errUnauthorized = errors.New("unauthorized")

// apiFromRequest creates an api client reusing the OAuth instance created for
// the same client, so results appear once the user has authorized via the UI.
func apiFromRequest(req *http.Request) (*API, error) {
	q := req.URL.Query()
	clientID, redirectURI := q.Get("clientid"), q.Get("redirecturi")

	if clientID == "" {
		return nil, errUnauthorized
	}

	log := util.NewLogger("viessmann").Redact(clientID)
	ctx := util.WithLogger(context.Background(), log)

	ts, err := NewOAuth(ctx, clientID, redirectURI, q.Get("gateway_serial"))
	if err != nil {
		return nil, err
	}

	// no values until the user has authorized
	if _, err := ts.Token(); err != nil {
		return nil, errUnauthorized
	}

	return NewAPI(log, ApiURI, ts), nil
}

// equipment lists the account's installations including their gateways.
func equipment(req *http.Request) ([]Installation, error) {
	api, err := apiFromRequest(req)
	if err != nil {
		return nil, err
	}

	return api.Installations()
}

// label describes the installation by its description or, if unset, its address.
func (inst Installation) label() string {
	if desc := strings.TrimSpace(inst.Description); desc != "" {
		return desc
	}

	street := strings.TrimSpace(inst.Address.Street + " " + inst.Address.HouseNumber)
	return strings.Trim(street+", "+inst.Address.City, ", ")
}

// equipmentOptions lists the installations or, with gateways set, the deduplicated
// gateways of the given installations, described by location and gateway type.
// All options are valid choices, match keeps the single-item auto-fill.
func equipmentOptions(installations []Installation, gateways bool) []service.Option {
	res := []service.Option{}

	for _, inst := range installations {
		if !gateways {
			serials := make([]string, 0, len(inst.Gateways))
			for _, gw := range inst.Gateways {
				if serial := strings.TrimSpace(gw.Serial); serial != "" {
					serials = append(serials, serial)
				}
			}
			res = append(res, service.Option{Value: strconv.Itoa(inst.ID), Label: inst.label(), Hint: strings.Join(serials, ", "), Match: true})
			continue
		}

		for _, gw := range inst.Gateways {
			if !slices.ContainsFunc(res, func(o service.Option) bool { return o.Value == gw.Serial }) {
				res = append(res, service.Option{Value: gw.Serial, Label: strings.TrimSpace(gw.GatewayType), Hint: inst.label(), Match: true})
			}
		}
	}

	slices.SortFunc(res, func(a, b service.Option) int { return cmp.Compare(a.Value, b.Value) })

	return res
}

// getEquipment lists the account's installations or, with detail=gateways,
// their gateways, driving the respective selection in the template.
func getEquipment(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	res := []service.Option{}
	defer func() { _ = json.NewEncoder(w).Encode(res) }()

	installations, err := equipment(req)
	if err != nil {
		// unexpected errors are worth logging, missing authorization is not
		if !errors.Is(err, errUnauthorized) {
			util.NewLogger("viessmann").ERROR.Println(err)
		}
		return
	}

	res = equipmentOptions(installations, req.URL.Query().Get("detail") == "gateways")
}

// measurementFeatures are the data points the charger template's optional
// measurements readers require. Availability is device-specific (One Base/E3
// device generation).
var measurementFeatures = []string{
	"heating.power.consumption.current",
	"heating.dhw.sensors.temperature.dhwCylinder",
	"heating.dhw.temperature.main",
}

// hasMeasurements reports whether the device provides all data points the
// optional measurements readers require.
func hasMeasurements(features []Feature) bool {
	for _, name := range measurementFeatures {
		if !slices.ContainsFunc(features, func(f Feature) bool { return f.Feature == name }) {
			return false
		}
	}
	return true
}

// getMeasurements reports whether the device provides the optional power and
// temperature data points, pre-setting the measurements toggle in the config UI.
func getMeasurements(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	res := []string{}
	defer func() { _ = json.NewEncoder(w).Encode(res) }()

	api, err := apiFromRequest(req)
	if err != nil {
		// unexpected errors are worth logging, missing authorization is not
		if !errors.Is(err, errUnauthorized) {
			util.NewLogger("viessmann").ERROR.Println(err)
		}
		return
	}

	q := req.URL.Query()
	features, err := api.Features(q.Get("installation_id"), q.Get("gateway_serial"), q.Get("device_id"))
	if err != nil {
		util.NewLogger("viessmann").ERROR.Println(err)
		return
	}

	res = []string{strconv.FormatBool(hasMeasurements(features))}
}
