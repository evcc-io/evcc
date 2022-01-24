package mercedes

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// const BaseURI = "https://api.mercedes-benz.com/vehicledata_tryout/v2"

// BaseURI is the Mercedes api base URI
const BaseURI = "https://api.mercedes-benz.com/vehicledata/v2"

// API is the Mercedes api client
type API struct {
	*request.Helper
	api.ProviderLogin
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:        request.NewHelper(log),
		ProviderLogin: identity,
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
	v.Client = oauth2.NewClient(ctx, identity)

	return v
}

// SoC implements the /soc response
func (v *API) SoC(vin string) (EVResponse, error) {
	var res EVResponse

	uri := fmt.Sprintf("%s/vehicles/%s/resources/soc", BaseURI, vin)
	err := v.GetJSON(uri, &res)

	return res, err
}

// Range implements the /rangeelectric response
func (v *API) Range(vin string) (EVResponse, error) {
	var res EVResponse

	uri := fmt.Sprintf("%s/vehicles/%s/resources/rangeelectric", BaseURI, vin)
	err := v.GetJSON(uri, &res)

	return res, err
}
