package porsche

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// API is the Porsche Connect (PPA) backend client.
type API struct {
	*request.Helper
}

// NewAPI creates a new Porsche Connect API client authenticated by identity.
func NewAPI(log *util.Logger, identity oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &transport.Decorator{
		Base: &oauth2.Transport{
			Source: identity,
			Base:   v.Client.Transport,
		},
		Decorator: transport.DecorateHeaders(map[string]string{
			"User-Agent":  UserAgent,
			"X-Client-ID": XClientID,
		}),
	}

	return v
}

// Vehicles returns the list of vehicles on the account.
func (v *API) Vehicles() ([]Vehicle, error) {
	var res []Vehicle
	uri := ApiURI + "/connect/v1/vehicles"
	err := v.GetJSON(uri, &res)
	return res, err
}

// Status returns the stored measurement overview for the given vehicle.
func (v *API) Status(vin string) (StatusResponse, error) {
	return v.overview(vin, "")
}

// WakeUp requests a live measurement overview from the vehicle, see CJNE/pyporscheconnectapi get_current_overview
func (v *API) WakeUp(vin string) (StatusResponse, error) {
	return v.overview(vin, "&wakeUpJob="+uuid.NewString())
}

func (v *API) overview(vin, query string) (StatusResponse, error) {
	var res StatusResponse
	mf := "mf=" + strings.Join(Measurements, "&mf=")
	uri := fmt.Sprintf("%s/connect/v1/vehicles/%s?%s%s", ApiURI, vin, mf, query)
	err := v.GetJSON(uri, &res)
	return res, err
}
