package ford

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

const ApiURI = "https://usapi.cv.ford.com"

// API is the Ford api client
type API struct {
	*request.Helper
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, ts oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &transport.Decorator{
		Decorator: func(req *http.Request) error {
			token, err := ts.Token()
			if err == nil {
				for k, v := range map[string]string{
					"Content-type":   request.JSONContent,
					"User-Agent":     "FordPass/5 CFNetwork/1333.0.4 Darwin/21.5.0",
					"locale":         "de-DE",
					"Application-Id": ApplicationID,
					"Auth-Token":     token.AccessToken,
					"CountryCode":    "DEU",
				} {
					req.Header.Set(k, v)
				}
			}
			return err
		},
		Base: v.Client.Transport,
	}

	return v
}

// Vehicles returns the list of user vehicles
func (v *API) Vehicles() ([]string, error) {
	var res []string

	data := map[string]string{
		"dashboardRefreshRequest": "All",
	}

	uri := fmt.Sprintf("%s/api/expdashboard/v1/details", TokenURI)

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		var resp VehiclesResponse
		if err = v.DoJSON(req, &resp); err == nil {
			for _, v := range resp.UserVehicles.VehicleDetails {
				res = append(res, v.VIN)
			}
		}
	}

	return res, err
}

// Status performs a /status request
func (v *API) Status(vin string) (StatusResponse, error) {
	uri := fmt.Sprintf("%s/api/vehicles/v5/%s/status", ApiURI, vin)

	var res StatusResponse
	err := v.GetJSON(uri, &res)

	return res, err
}

// RefreshResult retrieves a refresh result using /statusrefresh
func (v *API) RefreshResult(vin, refreshId string) (StatusResponse, error) {
	var res StatusResponse

	uri := fmt.Sprintf("%s/api/vehicles/v5/%s/statusrefresh/%s", ApiURI, vin, refreshId)
	err := v.GetJSON(uri, &res)

	return res, err
}

// RefreshRequest requests status refresh tracked by commandId
func (v *API) RefreshRequest(vin string) (string, error) {
	var resp struct {
		CommandId string
	}

	uri := fmt.Sprintf("%s/api/vehicles/v5/%s/status", ApiURI, vin)
	req, err := http.NewRequest(http.MethodPut, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	if err == nil && resp.CommandId == "" {
		err = errors.New("refresh failed")
	}

	return resp.CommandId, err
}

// WakeUp performs a wakeup request
func (v *API) WakeUp(vin string) error {
	uri := fmt.Sprintf("%s/api/dashboard/v1/users/vehicles?wakeupVin=%s", TokenURI, vin)

	_, err := v.GetBody(uri)

	return err
}
