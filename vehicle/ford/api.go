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

// API is the VW api client
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
					"Content-type":   "application/json",
					"Application-Id": "71A3AD0A-CF46-4CCF-B473-FC7FE5BC4592",
					"Auth-Token":     token.AccessToken,
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
	var resp VehiclesResponse
	var vehicles []string

	err := v.GetJSON(VehicleListURI, &resp)
	if err == nil {
		for _, v := range resp.Vehicles.Values {
			vehicles = append(vehicles, v.VIN)
		}
	}

	return vehicles, err
}

// Status performs a /status request to the Ford API and triggers a refresh if
// the received status is too old
func (v *API) Status(vin string) (StatusResponse, error) {
	// follow up requested refresh
	// if v.refreshId != "" {
	// 	return v.refreshResult()
	// }

	// otherwise start normal workflow
	uri := fmt.Sprintf("%s/api/vehicles/v3/%s/status", ApiURI, vin)

	var res StatusResponse
	err := v.GetJSON(uri, &res)

	// if err == nil {
	// 	var lastUpdate time.Time
	// 	lastUpdate, err = time.Parse(TimeFormat, res.StatusResponse.LastRefresh)

	// 	if elapsed := time.Since(lastUpdate); err == nil && elapsed > expiry {
	// 		v.log.DEBUG.Printf("vehicle status is outdated (age %v > %v), requesting refresh", elapsed, expiry)

	// 		if err = v.refreshRequest(); err == nil {
	// 			err = api.ErrMustRetry
	// 		}
	// 	}
	// }

	return res, err
}

// RefreshResult triggers an update if not already in progress, otherwise gets result
func (v *API) RefreshResult(vin, refreshId string) (StatusResponse, error) {
	var res StatusResponse

	uri := fmt.Sprintf("%s/api/vehicles/v3/%s/statusrefresh/%s", ApiURI, vin, refreshId)
	err := v.GetJSON(uri, &res)

	// // update successful and completed
	// if err == nil && res.Status == 200 {
	// 	v.refreshId = ""
	// 	return res, nil
	// }

	// // update still in progress, keep retrying
	// if time.Since(v.refreshTime) < RefreshTimeout {
	// 	return res, api.ErrMustRetry
	// }

	// // give up
	// v.refreshId = ""
	// if err == nil {
	// 	err = api.ErrTimeout
	// }

	return res, err
}

// RefreshRequest requests status refresh tracked by commandId
func (v *API) RefreshRequest(vin string) (string, error) {
	var resp struct {
		CommandId string
	}

	uri := fmt.Sprintf("%s/api/vehicles/v2/%s/status", ApiURI, vin)
	req, err := http.NewRequest(http.MethodPut, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	if err == nil {
		// 	v.refreshId = resp.CommandId
		// 	v.refreshTime = time.Now()

		if resp.CommandId == "" {
			err = errors.New("refresh failed")
		}
	}

	return resp.CommandId, err
}
