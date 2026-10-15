package myskoda

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// https://public.api.connect.skoda-auto.cz/docs/myskoda-public-api.yaml

const (
	BaseURI    = "https://public.api.connect.skoda-auto.cz/api/v1"
	SandboxURI = "https://public.test-api.connect.skoda-auto.cz/api/v1"
)

const (
	ActionStart = "start"
	ActionStop  = "stop"
)

// API is the MyŠkoda public api client
type API struct {
	*request.Helper
	uri string
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, uri, apiKey string) *API {
	v := &API{
		Helper: request.NewHelper(log),
		uri:    uri,
	}

	v.Client.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"X-API-Key": apiKey,
		}),
		Base: v.Client.Transport,
	}

	return v
}

// Vehicle returns the vehicle state, limited to the given parts
func (v *API) Vehicle(vin string, include ...string) (VehicleResponse, error) {
	var res VehicleResponse

	uri := fmt.Sprintf("%s/vehicles/%s", v.uri, vin)
	if len(include) > 0 {
		uri += "?include=" + strings.Join(include, ",")
	}

	err := v.GetJSON(uri, &res)
	return res, err
}

// ChargeAction starts or stops charging
func (v *API) ChargeAction(vin, action string) error {
	uri := fmt.Sprintf("%s/vehicles/%s/charging/%s", v.uri, vin, action)

	req, err := request.New(http.MethodPost, uri, nil, request.JSONEncoding)
	if err == nil {
		_, err = v.DoBody(req) // returns 202 with empty body
	}
	return err
}
