package aiways

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// https://github.com/snaptec/openWB/blob/master/modules/soc_aiways/aiways_get_soc.py

const URI = "https://coiapp-api-eu.ai-ways.com:10443"

// API implements the Aiways api.
type API struct {
	*request.Helper
	user, password string
}

// New creates a new BlueLink API
func NewAPI(log *util.Logger, user, password string) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}

	v.Client.Transport = &transport.Decorator{
		Base:      v.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			// "apikey": EmobilityOAuth2Config.ClientID,
		}),
	}

	return v
}

type Vehicle struct {
	VIN, VehicleName, VehicleID string
}

func (v *API) Vehicles() ([]Vehicle, error) {
	var res struct {
		Message string
		Data    any
	}

	data := struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}{
		Account:  v.user,
		Password: v.password,
	}

	uri := fmt.Sprintf("%s/aiways-passport-service/passport/login/password", URI)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	if err == nil {
		if err = v.DoJSON(req, &res); err == nil && res.Data == nil {
			err = errors.New(res.Message)
		}
	}

	return nil, err
}
