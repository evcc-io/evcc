package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// EquipmentURI is the Viessmann IoT API base used for equipment discovery
var EquipmentURI = "https://api.viessmann-climatesolutions.com/iot/v2"

var viessmannServiceMux = http.NewServeMux()

func init() {
	viessmannServiceMux.HandleFunc("GET /installations", getViessmannInstallations)
	viessmannServiceMux.HandleFunc("GET /gateways", getViessmannGateways)

	service.Register("viessmann", viessmannServiceMux)
}

type viessmannInstallation struct {
	ID       int `json:"id"`
	Gateways []struct {
		Serial string `json:"serial"`
	} `json:"gateways"`
}

// viessmannEquipmentList returns the account's installations including their
// gateways using the given, already authorized http client.
func viessmannEquipmentList(client *http.Client, base string) ([]viessmannInstallation, error) {
	resp, err := client.Get(base + "/equipment/installations?includeGateways=true")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("equipment installations: %s", resp.Status)
	}

	var res struct {
		Data []viessmannInstallation `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return res.Data, nil
}

// viessmannEquipment reuses the OAuth instance created for the same client, so
// results appear once the user has authorized via the UI.
func viessmannEquipment(req *http.Request) ([]viessmannInstallation, error) {
	q := req.URL.Query()
	clientID, redirectURI := q.Get("clientid"), q.Get("redirecturi")

	if clientID == "" {
		return nil, fmt.Errorf("missing clientid")
	}

	log := util.NewLogger("viessmann").Redact(clientID)
	ctx := util.WithLogger(context.Background(), log)

	ts, err := NewViessmannFromConfig(ctx, map[string]any{
		"clientid":       clientID,
		"redirecturi":    redirectURI,
		"gateway_serial": q.Get("gateway_serial"),
	})
	if err != nil {
		return nil, err
	}

	// no values until the user has authorized
	if _, err := ts.Token(); err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))

	return viessmannEquipmentList(oauth2.NewClient(ctx, ts), EquipmentURI)
}

// getViessmannInstallations lists the account's installation ids, driving
// installation selection in the template.
func getViessmannInstallations(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ids := []string{}
	defer func() { _ = json.NewEncoder(w).Encode(ids) }()

	installations, err := viessmannEquipment(req)
	if err != nil {
		return
	}

	for _, inst := range installations {
		ids = append(ids, strconv.Itoa(inst.ID))
	}
	slices.Sort(ids)
}

// getViessmannGateways lists the gateway serials of the account's
// installations, driving gateway selection in the template.
func getViessmannGateways(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	serials := []string{}
	defer func() { _ = json.NewEncoder(w).Encode(serials) }()

	installations, err := viessmannEquipment(req)
	if err != nil {
		return
	}

	for _, inst := range installations {
		for _, gw := range inst.Gateways {
			if !slices.Contains(serials, gw.Serial) {
				serials = append(serials, gw.Serial)
			}
		}
	}
	slices.Sort(serials)
}
