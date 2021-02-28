package meter

import (
	"fmt"
	"net/http"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

const discovergyAPI = "https://api.discovergy.com/public/v1"

func init() {
	registry.Add("discovergy", NewDiscovergyFromConfig)
}

// NewDiscovergyFromConfig creates a new configurable meter
func NewDiscovergyFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		User     string
		Password string
		Meter    string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("discgy")

	headers := make(map[string]string)
	if err := provider.AuthHeaders(log, provider.Auth{
		Type:     "Basic",
		User:     cc.User,
		Password: cc.Password,
	}, headers); err != nil {
		return nil, err
	}

	if cc.Meter == "" {
		req, err := request.New(http.MethodGet, fmt.Sprintf("%s/meters", discovergyAPI), nil, headers)
		if err == nil {
			var meters []struct {
				MeterID string `json:"meterId"`
			}

			client := request.NewHelper(log)
			if err = client.DoJSON(req, &meters); err == nil {
				if len(meters) == 1 {
					cc.Meter = meters[0].MeterID
				} else {
					err = fmt.Errorf("could not determine meter id: %v", meters)
				}
			}
		}

		if err != nil {
			return nil, err
		}
	}

	uri := fmt.Sprintf("%s/last_reading?meterId=%s", discovergyAPI, cc.Meter)
	power, err := provider.NewHTTP(log, http.MethodGet, uri, headers, "", false, ".values.power", 0.001)
	if err != nil {
		return nil, err
	}

	return NewConfigurable(power.FloatGetter())
}
