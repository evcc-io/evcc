package mercedes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

// const BaseURI = "https://api.mercedes-benz.com/vehicledata_tryout/v2"

// BaseURI is the Mercedes api base URI
const BaseURI = "https://api.mercedes-benz.com/vehicledata/v2"

// API is the Mercedes api client
type API struct {
	*request.Helper
	Identity *Identity

	updatedC chan struct{}
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, identity *Identity, updatedC chan struct{}) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		Identity: identity,

		updatedC: updatedC,
	}

	// authenticated http client with logging injected to the Mercedes client
	go func() {
		for range v.updatedC {
			log.TRACE.Println("update api client")
			ctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
			v.Client = identity.AuthConfig.Client(ctx, identity.Token())

			// TODO: hacky resetting all caches.
			provider.ResetCached()
		}
	}()

	return v
}

func (v *API) Update() chan struct{} {
	return v.updatedC
}

func (v *API) getJSON(uri string, res interface{}) error {
	if !v.Identity.LoggedIn() {
		return fmt.Errorf("token not valid")
	}

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
