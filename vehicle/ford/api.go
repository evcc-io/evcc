package ford

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// API is the VW api client
type API struct {
	*request.Helper
	ts oauth2.TokenSource
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, ts oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &oauth2.Transport{
		Source: identity,
		Base:   v.Client.Transport,
	}

	return v
}

// request is a helper to send API requests, sets header the Ford API expects
func (v *API) request(method, uri string) (*http.Request, error) {
	token, err := v.tokenSource.Token()

	var req *http.Request
	if err == nil {
		req, err = request.New(method, uri, nil, map[string]string{
			"Content-type":   "application/json",
			"Application-Id": "71A3AD0A-CF46-4CCF-B473-FC7FE5BC4592",
			"Auth-Token":     token.AccessToken,
		})
	}

	return req, err
}

// Vehicles returns the list of user vehicles
func (v *API) Vehicles() ([]string, error) {
	var resp VehiclesResponse

	req, err := v.request(http.MethodGet, VehicleListURI)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	var vehicles []string
	if err == nil {
		for _, v := range resp.Vehicles.Values {
			vehicles = append(vehicles, v.VIN)
		}
	}

	return vehicles, err
}

// Status performs a /status request to the Ford API and triggers a refresh if
// the received status is too old
func (v *API) Status(vin string) (res VehicleStatus, err error) {
	// follow up requested refresh
	// if v.refreshId != "" {
	// 	return v.refreshResult()
	// }

	// otherwise start normal workflow
	uri := fmt.Sprintf("%s/api/vehicles/v3/%s/status", ApiURI, vin)
	req, err := v.request(http.MethodGet, uri)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	// if err == nil {
	// 	var lastUpdate time.Time
	// 	lastUpdate, err = time.Parse(TimeFormat, res.VehicleStatus.LastRefresh)

	// 	if elapsed := time.Since(lastUpdate); err == nil && elapsed > expiry {
	// 		v.log.DEBUG.Printf("vehicle status is outdated (age %v > %v), requesting refresh", elapsed, expiry)

	// 		if err = v.refreshRequest(); err == nil {
	// 			err = api.ErrMustRetry
	// 		}
	// 	}
	// }

	return res, err
}

// refreshResult triggers an update if not already in progress, otherwise gets result
func (v *API) refreshResult() (res VehicleStatus, err error) {
	uri := fmt.Sprintf("%s/api/vehicles/v3/%s/statusrefresh/%s", ApiURI, v.vin, v.refreshId)

	var req *http.Request
	if req, err = v.request(http.MethodGet, uri); err == nil {
		err = v.DoJSON(req, &res)
	}

	// update successful and completed
	if err == nil && res.Status == 200 {
		v.refreshId = ""
		return res, nil
	}

	// update still in progress, keep retrying
	if time.Since(v.refreshTime) < RefreshTimeout {
		return res, api.ErrMustRetry
	}

	// give up
	v.refreshId = ""
	if err == nil {
		err = api.ErrTimeout
	}

	return res, err
}

// refreshRequest requests status refresh tracked by commandId
func (v *API) refreshRequest() error {
	var resp struct {
		CommandId string
	}

	uri := fmt.Sprintf("%s/api/vehicles/v2/%s/status", ApiURI, v.vin)
	req, err := v.request(http.MethodPut, uri)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	if err == nil {
		v.refreshId = resp.CommandId
		v.refreshTime = time.Now()

		if resp.CommandId == "" {
			err = errors.New("refresh failed")
		}
	}

	return err
}
