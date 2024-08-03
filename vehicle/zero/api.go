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
	user     string
	password string
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, user, password string) (*API, error) {
	var err error
	v := &API{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}
	return v, err
}

func (v *API) Vehicles() ([]Unit, error) {
	var res []Unit

	params := url.Values{
		"user":        {v.user},
		"pass":        {v.password},
		"commandname": {"get_units"},
		"format":      {"json"},
	}

	uri := fmt.Sprintf("%s?%s", BaseUrl, params.Encode())
	err := v.GetJSON(uri, &res)
	return res, err
}

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status(unitId string) (State, error) {
	var res []State

	params := url.Values{
		"user":        {v.user},
		"pass":        {v.password},
		"unitnumber":  {unitId},
		"commandname": {"get_last_transmit"},
		"format":      {"json"},
	}

	uri := fmt.Sprintf("%s?%s", BaseUrl, params.Encode())
	if err := v.GetJSON(uri, &res); err != nil {
		return State{}, err
	}

	return res[0], nil
}
