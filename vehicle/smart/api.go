package smart

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/mb"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.smart-eq

const ApiURI = "https://oneapp.microservice.smart.com/seqc/v0"

var OAuth2Config = &oauth2.Config{
	ClientID:    "70d89501-938c-4bec-82d0-6abb550b0825",
	RedirectURL: "https://oneapp.microservice.smart.com",
	Endpoint: oauth2.Endpoint{
		AuthURL:  mb.OAuthURI + "/as/authorization.oauth2",
		TokenURL: mb.OAuthURI + "/as/token.oauth2",
	},
	Scopes: []string{"openid", "profile", "email", "phone", "ciam-uid", "offline_access"},
}

type API struct {
	*request.Helper
}

func NewAPI(log *util.Logger, identity oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	// replace client transport with authenticated transport
	v.Client.Transport = &transport.Decorator{
		Base: &oauth2.Transport{
			Source: identity,
			Base:   v.Client.Transport,
		},
		Decorator: transport.DecorateHeaders(map[string]string{
			"accept":            "*/*",
			"accept-language":   "de-DE;q=1.0",
			"guid":              "280C6B55-F179-4428-88B6-E0CCF5C22A7C",
			"x-applicationname": OAuth2Config.ClientID,
		}),
	}

	return v
}

func (v *API) Vehicles() ([]string, error) {
	type vehicle struct {
		FIN string
	}

	var res struct {
		Authorizations, LicensePlates []vehicle
		Error                         string
		ErrorDescription              string `json:"error_description"`
	}

	uri := fmt.Sprintf("%s/users/current", ApiURI)
	err := v.GetJSON(uri, &res)
	if err != nil && res.Error != "" {
		err = fmt.Errorf("%s (%s): %w", res.Error, res.ErrorDescription, err)
	}

	vehicles := lo.Map(res.LicensePlates, func(v vehicle, _ int) string {
		return v.FIN
	})

	return vehicles, err
}

func (v *API) Status(vin string) (StatusResponse, error) {
	var res StatusResponse

	uri := fmt.Sprintf("%s/vehicles/%s/init-data?requestedData=BOTH&countryCode=DE&locale=de-DE", ApiURI, vin)
	err := v.GetJSON(uri, &res)

	if err != nil && res.Error != "" {
		err = fmt.Errorf("%s (%s): %w", res.Error, res.ErrorDescription, err)
	}

	return res, err
}

func (v *API) Refresh(vin string) (StatusResponse, error) {
	var res StatusResponse

	uri := fmt.Sprintf("%s/vehicles/%s/refresh-data", ApiURI, vin)
	err := v.GetJSON(uri, &res)

	if err != nil && res.Error != "" {
		err = fmt.Errorf("%s (%s): %w", res.Error, res.ErrorDescription, err)
	}

	return res, err
}
