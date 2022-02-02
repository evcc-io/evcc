package porsche

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

const (
	MobileApiURI = "https://api.ppa.porsche.com"
)

// API is an api.Vehicle implementation for Porsche PHEV cars
type MobileAPI struct {
	*request.Helper
}

// NewAPI creates a new vehicle
func NewMobileAPI(log *util.Logger, identity oauth2.TokenSource) *MobileAPI {
	v := &MobileAPI{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &transport.Decorator{
		Base: &oauth2.Transport{
			Source: identity,
			Base:   v.Client.Transport,
		},
		Decorator: transport.DecorateHeaders(map[string]string{
			"apikey":      OAuth2Config.ClientID,
			"x-client-id": "52064df8-6daa-46f7-bc9e-e3232622ab26",
		}),
	}

	return v
}

// Vehicles implements the vehicle list response
func (v *MobileAPI) Vehicles() ([]StatusResponseMobile, error) {
	var res []StatusResponseMobile
	uri := fmt.Sprintf("%s/app/connect/v1/vehicles", MobileApiURI)
	err := v.GetJSON(uri, &res)
	return res, err
}

// Status implements the vehicle status response
func (v *MobileAPI) Status(vin string) (StatusResponseMobile, error) {
	var res StatusResponseMobile
	uri := fmt.Sprintf("%s/app/connect/v1/vehicles/%s?mf=*", MobileApiURI, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}
