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

// Soc implements the /soc response
func (v *API) Soc(vin string) (EVResponse, error) {
	var res EVResponse

	uri := fmt.Sprintf("%s/vehicles/%s/resources/soc", v.BaseURI(), vin)
	err := v.GetJSON(uri, &res)
	if err != nil {
		res, err = v.allinOne(vin)
	}

	return res, err
}

// Range implements the /rangeelectric response
func (v *API) Range(vin string) (EVResponse, error) {
	var res EVResponse

	uri := fmt.Sprintf("%s/vehicles/%s/resources/rangeelectric", v.BaseURI(), vin)
	err := v.GetJSON(uri, &res)
	if err != nil {
		res, err = v.allinOne(vin)
	}

	return res, err
}

// Odometer implements the /odo response
func (v *API) Odometer(vin string) (EVResponse, error) {
	var res EVResponse

	uri := fmt.Sprintf("%s/vehicles/%s/resources/odo", v.BaseURI(), vin)
	err := v.GetJSON(uri, &res)
	if err != nil {
		res, err = v.allinOne(vin)
	}

	return res, err
}

// allinOne is a 'fallback' to gather both metrics range and soc.
// It is used in case for any reason the single endpoints return an error - which happend in the past.
func (v *API) allinOne(vin string) (EVResponse, error) {
	var res []EVResponse

	uri := fmt.Sprintf("%s/vehicles/%s/containers/electricvehicle", v.BaseURI(), vin)
	err := v.GetJSON(uri, &res)

	var evres EVResponse

	for _, r := range res {
		if r.Soc.Timestamp != 0 {
			evres.Soc = r.Soc
			continue
		}

		if r.RangeElectric.Timestamp != 0 {
			evres.RangeElectric = r.RangeElectric
		}

		if r.Odometer.Timestamp != 0 {
			evres.Odometer = r.Odometer
		}
	}

	return evres, err
}
