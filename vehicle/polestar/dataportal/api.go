package dataportal

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// BaseURL is the Polestar Data Portal M2M API base URL
const BaseURL = "https://pc-api.polestar.com/eu-north-1/data-portal/m2m"

// API is the Polestar Data Portal REST client
type API struct {
	*request.Helper
}

// NewAPI creates a Polestar Data Portal API client. The accountID is sent as
// the x-client-id header required by the vehicle and telemetry endpoints.
func NewAPI(log *util.Logger, ts oauth2.TokenSource, accountID string) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Transport = &oauth2.Transport{
		Source: ts,
		Base: &transport.Decorator{
			Decorator: transport.DecorateHeaders(map[string]string{
				"x-client-id": accountID,
			}),
			Base: v.Transport,
		},
	}

	return v
}

// Vehicles returns the VINs the authenticated caller is authorized to access
func (v *API) Vehicles() ([]string, error) {
	var res struct {
		Data []string `json:"data"`
	}
	err := v.get(BaseURL+"/v1/vehicles", &res)
	return res.Data, err
}

// Battery returns the battery telemetry for the given VIN
func (v *API) Battery(vin string) (Battery, error) {
	var res struct {
		Data Battery `json:"data"`
	}
	err := v.get(fmt.Sprintf("%s/v1/vehicles/%s/telemetry/battery", BaseURL, vin), &res)
	return res.Data, err
}

// Odometer returns the odometer telemetry for the given VIN
func (v *API) Odometer(vin string) (Odometer, error) {
	var res struct {
		Data Odometer `json:"data"`
	}
	err := v.get(fmt.Sprintf("%s/v1/vehicles/%s/telemetry/odometer", BaseURL, vin), &res)
	return res.Data, err
}

// TargetSoc returns the configured target state of charge for the given VIN
func (v *API) TargetSoc(vin string) (TargetSoc, error) {
	var res struct {
		Data TargetSoc `json:"data"`
	}
	err := v.get(fmt.Sprintf("%s/v1/vehicles/%s/charging/target-soc", BaseURL, vin), &res)
	return res.Data, err
}

// get wraps GetJSON and maps Data Portal error responses to evcc sentinel errors
func (v *API) get(uri string, res any) error {
	return mapError(v.GetJSON(uri, res))
}

// mapError translates Data Portal HTTP errors into evcc sentinel errors: a
// missing-data 404 becomes api.ErrNotAvailable, transient upstream failures
// become api.ErrMustRetry so util.Cached retries instead of backing off.
func mapError(err error) error {
	var se *request.StatusError
	if errors.As(err, &se) {
		switch {
		case se.HasStatus(http.StatusNotFound):
			return api.ErrNotAvailable
		case se.StatusCode() >= http.StatusInternalServerError:
			return fmt.Errorf("%w: %w", api.ErrMustRetry, err)
		}
	}
	return err
}
