package meter

import (
	"fmt"
	"net/http"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/thoas/go-funk"
)

const discovergyAPI = "https://api.discovergy.com/public/v1"

type discovergyConfig struct {
	User     string `validate:"required"`
	Password string `validate:"required" ui:",mask"`
	Meter    string `ui:"de=Zählernummer"`
}

func init() {
	fmt.Println("-- Discovergy --")
	registry.Add("discovergy", "Discovergy", NewDiscovergyFromConfig, discovergyConfig{})
}

// NewDiscovergyFromConfig creates a new configurable meter
func NewDiscovergyFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc discovergyConfig

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

	// find meter id, given id or serial
	meter, err := discovergyMeterID(log, cc.Meter, headers)
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("%s/last_reading?meterId=%s", discovergyAPI, meter)
	power, err := provider.NewHTTP(log, http.MethodGet, uri, headers, "", false, ".values.power", 0.001)
	if err != nil {
		return nil, err
	}

	return NewConfigurable(power.FloatGetter())
}

func discovergyMeterID(log *util.Logger, id string, headers map[string]string) (string, error) {
	type meter struct {
		MeterID      string `json:"meterId"`
		SerialNumber string `json:"fullSerialNumber"`
	}
	var meters []meter

	req, err := request.New(http.MethodGet, fmt.Sprintf("%s/meters", discovergyAPI), nil, headers)
	if err == nil {
		client := request.NewHelper(log)
		err = client.DoJSON(req, &meters)
	}

	if err == nil {
		if id == "" && len(meters) == 1 {
			return meters[0].MeterID, nil
		}

		for _, m := range meters {
			if id == m.MeterID || id == m.SerialNumber {
				return m.MeterID, nil
			}
		}

		err = fmt.Errorf("could not find meter, got: %v", funk.Map(meters, func(m meter) string {
			return m.SerialNumber
		}))
	}

	return "", err
}
