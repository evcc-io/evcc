package viessmann

import (
	"fmt"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Gateway is a Viessmann communication module.
type Gateway struct {
	Serial string `json:"serial"`
}

// Installation is a Viessmann installation including its gateways.
type Installation struct {
	ID       int       `json:"id"`
	Gateways []Gateway `json:"gateways"`
}

// API is the Viessmann IoT API client.
type API struct {
	*request.Helper
	uri string
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, uri string, ts oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
		uri:    uri,
	}

	v.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   v.Client.Transport,
	}

	return v
}

// Installations returns the account's installations including their gateways.
func (v *API) Installations() ([]Installation, error) {
	var res struct {
		Data []Installation `json:"data"`
	}

	err := v.GetJSON(v.uri+"/equipment/installations?includeGateways=true", &res)

	return res.Data, err
}

// Feature is a data point of a Viessmann device.
type Feature struct {
	Feature string `json:"feature"`
}

// Features returns the data points the given device provides. The list is
// device-specific and self-describing: a data point is supported iff it appears.
func (v *API) Features(installationID, gatewaySerial, deviceID string) ([]Feature, error) {
	var res struct {
		Data []Feature `json:"data"`
	}

	uri := fmt.Sprintf("%s/features/installations/%s/gateways/%s/devices/%s/features",
		v.uri, url.PathEscape(installationID), url.PathEscape(gatewaySerial), url.PathEscape(deviceID))

	err := v.GetJSON(uri, &res)

	return res.Data, err
}
