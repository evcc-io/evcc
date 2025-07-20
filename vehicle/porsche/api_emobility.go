package porsche

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// EmobilityAPI is the Porsche Emobility API
type EmobilityAPI struct {
	*request.Helper
}

// NewEmobilityAPI creates a new vehicle
func NewEmobilityAPI(log *util.Logger, identity oauth2.TokenSource) *EmobilityAPI {
	v := &EmobilityAPI{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &transport.Decorator{
		Base: &oauth2.Transport{
			Source: identity,
			Base:   v.Client.Transport,
		},
		Decorator: transport.DecorateHeaders(map[string]string{
			"apikey": ClientID,
		}),
	}

	return v
}

func (v *EmobilityAPI) Capabilities(vin string) (CapabilitiesResponse, error) {
	var res CapabilitiesResponse
	uri := fmt.Sprintf("%s/service-vehicle/vcs/capabilities/%s", ApiURI, vin)
	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"x-vrs-url-country":  "de",
		"x-vrs-url-language": "de_DE",
	})
	err := v.DoJSON(req, &res)
	return res, err
}

// Status implements the vehicle status response
func (v *EmobilityAPI) Status(vin, model string) (EmobilityResponse, error) {
	var res EmobilityResponse

	uri := fmt.Sprintf("%s/e-mobility/de/de_DE/%s/%s?timezone=Europe/Berlin", ApiURI, model, vin)
	err := v.GetJSON(uri, &res)
	if err != nil && res.PcckErrorMessage != "" {
		err = errors.New(res.PcckErrorMessage)
	}

	return res, err
}
