package mercedes

import (
	"context"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/provider"
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

	providerLogin *Login
	updatedC      chan struct{}
	lock          sync.Mutex
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, identity *Identity, providerLogin *Login, updatedC chan struct{}) *API {
	v := &API{
		Helper: request.NewHelper(log),

		providerLogin: providerLogin,
		updatedC:      updatedC,
		lock:          sync.Mutex{},
	}

	// authenticated http client with logging injected to the Mercedes client
	go func() {
		for range v.updatedC {
			log.TRACE.Println("update api client")

			v.lock.Lock()

			ctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
			v.Client = identity.AuthConfig.Client(ctx, v.providerLogin.Token())

			v.lock.Unlock()

			// TODO: hacky resetting all caches.
			provider.ResetCached()
		}
	}()

	return v
}

func (v *API) Update() chan struct{} {
	return v.updatedC
}

// SoC implements the /soc response
func (v *API) SoC(vin string) (EVResponse, error) {
	if !v.providerLogin.Valid() {
		return EVResponse{}, fmt.Errorf("invalid provider login")
	}

	var res EVResponse
	uri := fmt.Sprintf("%s/vehicles/%s/resources/soc", BaseURI, vin)

	v.lock.Lock()
	defer v.lock.Unlock()

	err := v.GetJSON(uri, &res)

	return res, err
}

// Range implements the /rangeelectric response
func (v *API) Range(vin string) (EVResponse, error) {
	if !v.providerLogin.Valid() {
		return EVResponse{}, fmt.Errorf("invalid provider login")
	}

	var res EVResponse
	uri := fmt.Sprintf("%s/vehicles/%s/resources/rangeelectric", BaseURI, vin)

	v.lock.Lock()
	defer v.lock.Unlock()

	err := v.GetJSON(uri, &res)

	return res, err
}
