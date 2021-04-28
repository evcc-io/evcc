package mercedes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

// BaseURI is the VW api base URI
// const BaseURI = "https://api.mercedes-benz.com/vehicledata_tryout/v2"
const BaseURI = "https://api.mercedes-benz.com/vehicledata/v2"

// API is the Mercedes api client
type API struct {
	*request.Helper
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	// authenticated http client with logging injected to the Mercedes client
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
	v.Client = identity.AuthConfig.Client(ctx, identity.Token())

	return v
}

func (v *API) getJSON(uri string, res interface{}) error {
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept": "application/json",
	})

	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return err
}

type EVResponse struct {
	SoC struct {
		Value     IntVal
		Timestamp int64
	}
	RangeElectric struct {
		Value     IntVal
		Timestamp int64
	}
}

// SoC implements the /soc response
func (v *API) SoC(vin string) (EVResponse, error) {
	var res EVResponse
	uri := fmt.Sprintf("%s/vehicles/%s/resources/soc", BaseURI, vin)
	err := v.getJSON(uri, &res)
	return res, err
}

// Range implements the /rangeelectric response
func (v *API) Range(vin string) (EVResponse, error) {
	var res EVResponse
	uri := fmt.Sprintf("%s/vehicles/%s/resources/rangeelectric", BaseURI, vin)
	err := v.getJSON(uri, &res)
	return res, err
}
