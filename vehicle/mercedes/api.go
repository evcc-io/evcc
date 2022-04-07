package mercedes

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	// BaseURI is the Mercedes api base URI
	BaseURI        = "https://api.mercedes-benz.com/vehicledata/v2"
	SandboxBaseURI = "https://api.mercedes-benz.com/vehicledata_tryout/v2"
)

// API is the Mercedes api client
type API struct {
	*request.Helper
	sandbox bool
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, identity *Identity, sandbox bool) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	// replace client transport with authenticated transport
	v.Client.Transport = &oauth2.Transport{
		Source: identity,
		Base:   v.Client.Transport,
	}

	return v
}

func (v *API) BaseURI() string {
	if v.sandbox {
		return SandboxBaseURI
	}
	return BaseURI
}

// SoC implements the /soc response
func (v *API) SoC(vin string) (EVResponse, error) {
	var res EVResponse

	uri := fmt.Sprintf("%s/vehicles/%s/resources/soc", v.BaseURI(), vin)
	err := v.GetJSON(uri, &res)

	return res, err
}

// Range implements the /rangeelectric response
func (v *API) Range(vin string) (EVResponse, error) {
	var res EVResponse

	uri := fmt.Sprintf("%s/vehicles/%s/resources/rangeelectric", v.BaseURI(), vin)
	err := v.GetJSON(uri, &res)

	return res, err
}
