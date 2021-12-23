package mb

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/thoas/go-funk"
	"golang.org/x/oauth2"
)

const ApiURI = "https://oneapp.microservice.smart.com/seqc/v0"

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
			// "user-agent":        "Device: iPhone 6; OS-version: iOS_12.5.1; App-Name: smart EQ control; App-Version: 3.0; Build: 202108260942; Language: de_DE",
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

	vehicles := funk.Map(res.LicensePlates, func(v vehicle) string {
		return v.FIN
	}).([]string)

	return vehicles, err
}

func (v *API) Call(vin string) error {
	// 	await this.requestClient({
	// 		method: "get",
	// 		url: "%s/vehicles/" + vin + "/init-data?requestedData=BOTH&countryCode=DE&locale=de-DE",
	// 		headers: {
	// 			accept: "*/*",
	// 			"accept-language": "de-DE;q=1.0",
	// 			authorization: "Bearer " + this.session.access_token,
	// 			"x-applicationname": "70d89501-938c-4bec-82d0-6abb550b0825",
	// 			"user-agent": "Device: iPhone 6; OS-version: iOS_12.5.1; App-Name: smart EQ control; App-Version: 3.0; Build: 202108260942; Language: de_DE",
	// 			guid: "280C6B55-F179-4428-88B6-E0CCF5C22A7C",
	// 		},
	// 	})
	// 		.then(async (res) => {
	// 			this.log.debug(JSON.stringify(res.data));
	// 			this.json2iob.parse(vin, res.data);
	// 		})
	// 		.catch((error) => {
	// 			this.log.error(error);
	// 			error.response && this.log.error(JSON.stringify(error.response.data));
	// 		});
	// }

	var res struct {
		Error            string
		ErrorDescription string `json:"error_description"`
	}

	uri := fmt.Sprintf("%s/vehicles/%s/init-data?requestedData=BOTH&countryCode=DE&locale=de-DE", ApiURI, vin)
	// req, err := request.New(http.MethodGet, uri, nil, map[string]string{
	// 	"accept":            "*/*",
	// 	"accept-language":   "de-DE;q=1.0",
	// 	"x-applicationname": "70d89501-938c-4bec-82d0-6abb550b0825",
	// 	"guid":              "280C6B55-F179-4428-88B6-E0CCF5C22A7C",
	// })
	// if err == nil {
	// 	if err = v.DoJSON(req, &res); err != nil && res.Error != "" {
	// 		err = fmt.Errorf("%s (%s): %w", res.Error, res.ErrorDescription, err)
	// 	}
	// }

	err := v.GetJSON(uri, &res)
	if err != nil && res.Error != "" {
		err = fmt.Errorf("%s (%s): %w", res.Error, res.ErrorDescription, err)
	}

	return err
}
