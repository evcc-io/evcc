package zero

import (
	"fmt"
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

// API is an api.Vehicle implementation for Zero Motorcycles
type API struct {
	*request.Helper
	identity *Identity
	log      *util.Logger
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		identity: identity,
		log:      log,
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

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status() (ZeroState, error) {
	var states []ZeroState
	var res ZeroState

	params := url.Values{
		"user":        {v.identity.user},
		"pass":        {v.identity.password},
		"unitnumber":  {v.identity.unitId},
		"commandname": {"get_last_transmit"},
		"format":      {"json"},
	}

	uri := fmt.Sprintf("%s?%s", BaseUrl, params.Encode())
	err := v.GetJSON(uri, &states)
	if err == nil {
		return states[0], nil
	}
	return res, err
}
