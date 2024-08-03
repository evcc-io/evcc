package zero

import (
	"fmt"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const BaseUrl = "https://mongol.brono.com/mongol/api.php"

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
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		identity: identity,
	}

	return v
}

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status() (ZeroState, error) {
	var res []ZeroState

	params := url.Values{
		"user":        {v.identity.user},
		"pass":        {v.identity.password},
		"unitnumber":  {v.identity.unitId},
		"commandname": {"get_last_transmit"},
		"format":      {"json"},
	}

	uri := fmt.Sprintf("%s?%s", BaseUrl, params.Encode())
	if err := v.GetJSON(uri, &res); err != nil {
		return ZeroState{}, err
	}

	return res[0], nil
}
