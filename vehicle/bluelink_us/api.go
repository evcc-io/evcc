package bluelink_us

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

const (
	ApiURL           = "/ac/v2/"
	EnrollmentURL    = "enrollment/details/%s"
	VehicleStatusURL = "rcs/rvs/vehicleStatus"
	PositionURL      = "rcs/rfc/findMyCar"
	ChargeStartURL   = "evc/charge/start"
	ChargeStopURL    = "evc/charge/stop"
)

type APIConfig struct {
	User           string
	Pin            string
	RegistrationID string
	VIN            string
	Generation     string
}

type API struct {
	*request.Helper
	baseURI string
	user    string
}

func NewAPI(log *util.Logger, ts oauth2.TokenSource, cfg APIConfig) *API {
	v := &API{
		Helper:  request.NewHelper(log),
		baseURI: BaseURL + ApiURL,
		user:    cfg.User,
	}

	// API can be slow
	v.Client.Timeout = 30 * time.Second

	// Add transport decorator for auth headers
	v.Client.Transport = &transport.Decorator{
		Decorator: func(req *http.Request) error {
			token, err := ts.Token()
			if err != nil {
				return err
			}

			for k, v := range BaseHeaders() {
				req.Header.Set(k, v)
			}
			req.Header.Set("username", cfg.User)
			req.Header.Set("accessToken", token.AccessToken)

			// Vehicle-specific headers
			if cfg.Pin != "" {
				req.Header.Set("blueLinkServicePin", cfg.Pin)
			}
			if cfg.RegistrationID != "" {
				req.Header.Set("registrationId", cfg.RegistrationID)
			}
			if cfg.Generation != "" {
				req.Header.Set("gen", cfg.Generation)
			}
			if cfg.VIN != "" {
				req.Header.Set("vin", cfg.VIN)
			}

			return nil
		},
		Base: v.Client.Transport,
	}

	return v
}

func (v *API) Vehicles() ([]Vehicle, error) {
	var res EnrollmentResponse

	uri := v.baseURI + fmt.Sprintf(EnrollmentURL, v.user)
	err := v.GetJSON(uri, &res)

	vehicles := make([]Vehicle, 0, len(res.EnrolledVehicleDetails))
	for _, entry := range res.EnrolledVehicleDetails {
		vehicles = append(vehicles, entry.VehicleDetails)
	}

	return vehicles, err
}

func (v *API) Status() (VehicleStatus, error) {
	var res VehicleStatusResponse

	uri := v.baseURI + VehicleStatusURL
	err := v.GetJSON(uri, &res)

	return res.VehicleStatus, err
}

func (v *API) Position() (PositionResponse, error) {
	var res PositionResponse

	uri := v.baseURI + PositionURL
	err := v.GetJSON(uri, &res)

	return res, err
}

func (v *API) ChargeStart() error {
	uri := v.baseURI + ChargeStartURL

	req, err := request.New(http.MethodPost, uri, nil, nil)
	if err != nil {
		return err
	}

	resp, err := v.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
	return err
}

func (v *API) ChargeStop() error {
	uri := v.baseURI + ChargeStopURL

	req, err := request.New(http.MethodPost, uri, nil, nil)
	if err != nil {
		return err
	}

	resp, err := v.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
	return err
}
