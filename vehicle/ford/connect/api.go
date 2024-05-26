package connect

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
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

	v.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   v.Client.Transport,
	}

	return v
}

const TokenURI = "https"

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

func (v *API) Status(vin string) (InformationResponse, error) {
	var res InformationResponse

	uri := fmt.Sprintf("%s/api/vehicles/v4/%s/status", ApiURI, "VIN")

	req, err := request.New(http.MethodGet, uri, nil, request.AcceptJSON)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}
