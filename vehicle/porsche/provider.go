package porsche

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type StatusResponse struct {
	CarControlData struct {
		BatteryLevel struct {
			Unit  string
			Value float64
		}
		Mileage struct {
			Unit  string
			Value float64
		}
		RemainingRanges struct {
			ElectricalRange struct {
				Distance struct {
					Unit  string
					Value float64
				}
			}
		}
	}
}

// Provider is an api.Vehicle implementation for Porsche PHEV cars
type Provider struct {
	log *util.Logger
	*request.Helper
	token    oauth2.Token
	identity *Identity
	statusG  func() (StatusResponse, error)
}

// NewProvider creates a new vehicle
func NewProvider(log *util.Logger, identity *Identity, token oauth2.Token, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		log:      log,
		Helper:   request.NewHelper(log),
		token:    token,
		identity: identity,
	}

	impl.statusG = provider.NewCached[StatusResponse](func() (StatusResponse, error) {
		return impl.status(vin)
	}, cache).Get

	return impl
}

func (v *Provider) request(uri string) (*http.Request, error) {
	if v.token.AccessToken == "" || time.Since(v.token.Expiry) > 0 {
		accessTokens, err := v.identity.Login()
		if err != nil {
			return nil, err
		}
		v.token = accessTokens.Token
	}

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", v.token.AccessToken),
	})

	return req, err
}

// Status implements the vehicle status response
func (v *Provider) status(vin string) (StatusResponse, error) {
	uri := fmt.Sprintf("https://connect-portal.porsche.com/core/api/v3/de/de_DE/vehicles/%s", vin)
	req, err := v.request(uri)
	if err != nil {
		return StatusResponse{}, err
	}

	var pr StatusResponse
	err = v.DoJSON(req, &pr)

	return pr, err
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		return res.CarControlData.BatteryLevel.Value, nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err == nil {
		return int64(res.CarControlData.RemainingRanges.ElectricalRange.Distance.Value), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		return res.CarControlData.Mileage.Value, nil
	}

	return 0, err
}
