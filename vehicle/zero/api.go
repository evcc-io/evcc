package zero

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// See https://www.electricmotorcycleforum.com/boards/index.php?topic=9520.0
// API is quite simple
// 1 Acquire unit id(s) by calling
// https://mongol.brono.com/mongol/api.php?commandname=get_units&format=json&user=yourusername&pass=yourpass
// 2 Query last dataset
// https://mongol.brono.com/mongol/api.php?commandname=get_last_transmit&format=json&user=yourusername&pass=yourpass&unitnumber=0000000

// API is an api.Vehicle implementation for SAIC cars
type API struct {
	*request.Helper
	identity *Identity
	Logger   *util.Logger
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		identity: identity,
		Logger:   log,
	}

	return v
}

/*
Vehicles implements returns the /user/vehicles api

	func (v *API) Vehicles() ([]Vehicle, error) {
		var res []Vehicle
		uri := fmt.Sprintf("%s/eadrax-vcs/v4/vehicles?apptimezone=120&appDateTime=%d", regions[v.region].CocoApiURI, time.Now().UnixMilli())
		err := v.GetJSON(uri, &res)
		return res, err
	}
*/
func createRequest(params *url.Values) (*http.Request, error) {

	req, err := http.NewRequest("GET", BASE_URL_P, nil)
	if err != nil {
		return nil, err
	}

	params.Set("format", "json")
	req.URL.RawQuery = params.Encode()

	return req, err
}

func handleResponse(resp *http.Response) (*[]byte, error) {

	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		var answer ErrorAnswer
		err = json.Unmarshal(body, &answer)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(answer.Error)
	}
	return &body, nil
}

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status() (ZeroState, error) {
	var states []ZeroState
	var dummy ZeroState
	var resp *http.Response

	params := url.Values{
		"user":        {v.identity.User},
		"pass":        {v.identity.Password}, // Shold be Sha1 encoded
		"unitnumber":  {v.identity.UnitId},
		"commandname": {"get_last_transmit"},
	}

	req, err := createRequest(&params)
	if err != nil {
		return dummy, err
	}

	resp, err = v.Do(req)
	defer resp.Body.Close()

	if err != nil {
		return dummy, err
	}

	var body *[]byte
	body, err = handleResponse(resp)
	if err != nil {
		return dummy, err
	} else {
		err = json.Unmarshal(*body, &states)
		if err != nil {
			return dummy, err
		}
	}
	return states[0], err
}
