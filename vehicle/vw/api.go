package vw

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// DefaultBaseURI is the VW api base URI
const DefaultBaseURI = "https://msg.volkswagen.de/fs-car"

// RegionAPI is the VW api used for determining the home region
const RegionAPI = "https://mal-1a.prd.ece.vwg-connect.com/api"

// API is the VW api client
type API struct {
	*request.Helper
	brand, country string
	baseURI        string
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, ts oauth2.TokenSource, brand, country string) *API {
	v := &API{
		Helper:  request.NewHelper(log),
		brand:   brand,
		country: country,
		baseURI: DefaultBaseURI,
	}

	v.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   v.Client.Transport,
	}

	return v
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles() ([]string, error) {
	var res VehiclesResponse
	uri := fmt.Sprintf("%s/usermanagement/users/v1/%s/%s/vehicles", v.baseURI, v.brand, v.country)
	err := v.GetJSON(uri, &res)
	if err != nil && res.Error != nil {
		err = res.Error.Error()
	}
	return res.UserVehicles.Vehicle, err
}

// HomeRegion updates the home region for the given vehicle
func (v *API) HomeRegion(vin string) error {
	var res HomeRegion
	uri := fmt.Sprintf("%s/cs/vds/v1/vehicles/%s/homeRegion", RegionAPI, vin)

	err := v.GetJSON(uri, &res)
	if err == nil {
		if api := res.HomeRegion.BaseURI.Content; strings.HasPrefix(api, "https://mal-3a.prd.eu.dp.vwg-connect.com") {
			api = "https://fal" + strings.TrimPrefix(api, "https://mal")
			api = strings.TrimSuffix(api, "/api") + "/fs-car"
			v.baseURI = api
		}
	} else if res.Error != nil {
		err = res.Error.Error()
	}

	return err
}

// RolesRights implements the /rolesrights/operationlist response
func (v *API) RolesRights(vin string) (RolesRights, error) {
	var res RolesRights
	uri := fmt.Sprintf("%s/rolesrights/operationlist/v3/vehicles/%s", RegionAPI, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}

// Status implements the /status response
func (v *API) Status(vin string) (string, error) {
	var res json.RawMessage
	uri := fmt.Sprintf("%s/bs/vsr/v1/vehicles/%s", RegionAPI, vin)
	err := v.GetJSON(uri, &res)
	return string(res), err
}

// Charger implements the /charger response
func (v *API) Charger(vin string) (ChargerResponse, error) {
	var res ChargerResponse
	uri := fmt.Sprintf("%s/bs/batterycharge/v1/%s/%s/vehicles/%s/charger", v.baseURI, v.brand, v.country, vin)
	err := v.GetJSON(uri, &res)
	if err != nil && res.Error != nil {
		err = res.Error.Error()
	}
	return res, err
}

// Climater implements the /climater response
func (v *API) Climater(vin string) (ClimaterResponse, error) {
	var res ClimaterResponse
	uri := fmt.Sprintf("%s/bs/climatisation/v1/%s/%s/vehicles/%s/climater", v.baseURI, v.brand, v.country, vin)
	err := v.GetJSON(uri, &res)
	if err != nil && res.Error != nil {
		err = res.Error.Error()
	}
	return res, err
}

const (
	ActionCharge      = "batterycharge"
	ActionChargeStart = "start"
	ActionChargeStop  = "stop"
)

type actionDefinition struct {
	contentType string
	appendix    string
}

var actionDefinitions = map[string]actionDefinition{
	ActionCharge: {
		"application/vnd.vwg.mbb.ChargerAction_v1_0_0+xml",
		"charger/actions",
	},
}

// Action implements vehicle actions
func (v *API) Action(vin, action, value string) error {
	def := actionDefinitions[action]

	uri := fmt.Sprintf("%s/bs/%s/v1/%s/%s/vehicles/%s/%s", v.baseURI, action, v.brand, v.country, vin, def.appendix)
	body := "<?xml version=\"1.0\" encoding=\"UTF-8\" ?><action><type>" + value + "</type></action>"

	req, err := request.New(http.MethodPost, uri, strings.NewReader(body), map[string]string{
		"Content-type": def.contentType,
	})

	if err == nil {
		var resp *http.Response
		if resp, err = v.Do(req); err == nil {
			resp.Body.Close()
		}
	}

	return err
}

// Any implements any api response
func (v *API) Any(base, vin string) (interface{}, error) {
	var res interface{}
	uri := fmt.Sprintf("%s/"+strings.TrimLeft(base, "/"), v.baseURI, v.brand, v.country, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}
