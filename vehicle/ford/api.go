package ford

import (
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
			if err != nil {
				return err
			}

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

			return nil
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
