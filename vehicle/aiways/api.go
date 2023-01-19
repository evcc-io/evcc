package aiways

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// https://github.com/snaptec/openWB/blob/master/modules/soc_aiways/aiways_get_soc.py

const URI = "https://coiapp-api-eu.ai-ways.com:10443"

// API implements the Aiways api
type API struct {
	*request.Helper
	identity TokenProvider
}

// New creates a new Aiways API
func NewAPI(log *util.Logger, identity TokenProvider) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		identity: identity,
	}

	return v
}

// func (v *API) Vehicles() ([]Vehicle, error) {
// }

func (v *API) Status(user int64, vin string) (StatusResponse, error) {
	var res StatusResponse

	data2 := struct {
		UserId int64  `json:"userId"`
		VIN    string `json:"vin"`
	}{
		UserId: user,
		VIN:    vin,
	}

	uri := fmt.Sprintf("%s/app/vc/getCondition", URI)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data2), map[string]string{
		"Content-Type": request.JSONContent,
		"Accept":       request.JSONContent,
		"Token":        v.identity.Token(),
	})

	if err == nil {
		if err = v.DoJSON(req, &res); err == nil && res.Data == nil {
			err = errors.New(res.Message)
		}
	}

	return res, err
}
