package mercedes

import (
	"fmt"
	"net/http"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// BaseURI is the VW api base URI
const BaseURI = "https://api.mercedes-benz.com/vehicledata_tryout/v2"

// API is the VW api client
type API struct {
	*request.Helper
	identity *Identity
	baseURI  string
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		identity: identity,
	}
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
		Value     int
		Timestamp int64
	}
	RangeElectric struct {
		Value     int
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
