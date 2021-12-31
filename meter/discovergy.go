package meter

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/provider/pipeline"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/thoas/go-funk"
)

const discovergyAPI = "https://api.discovergy.com/public/v1"

func init() {
	registry.Add("discovergy", NewDiscovergyFromConfig)
}

type discovergyMeter struct {
	MeterID          string `json:"meterId"`
	SerialNumber     string `json:"serialNumber"`
	FullSerialNumber string `json:"fullSerialNumber"`
}

// NewDiscovergyFromConfig creates a new configurable meter
func NewDiscovergyFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		User     string
		Password string
		Meter    string
		Scale    float64
	}{
		Scale: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	basicAuth := transport.BasicAuthHeader(cc.User, cc.Password)

	log := util.NewLogger("discgy").Redact(cc.User, cc.Password, cc.Meter, basicAuth)

	client := request.NewHelper(log)
	client.Transport = transport.BasicAuth(cc.User, cc.Password, client.Transport)

	var meters []discovergyMeter
	if err := client.GetJSON(fmt.Sprintf("%s/meters", discovergyAPI), &meters); err != nil {
		return nil, err
	}

	var meterID string
	if cc.Meter != "" {
		for _, m := range meters {
			if matchesIdentifier(cc.Meter, m) {
				meterID = m.MeterID
				break
			}
		}
	} else if len(meters) == 1 {
		meterID = meters[0].MeterID
	}

	if meterID == "" {
		return nil, fmt.Errorf("could not determine meter id: %v", funk.Map(meters, func(m discovergyMeter) string {
			return m.FullSerialNumber
		}))
	}

	uri := fmt.Sprintf("%s/last_reading?meterId=%s", discovergyAPI, meterID)
	power, err := provider.NewHTTP(log, http.MethodGet, uri, false, 0.001*cc.Scale, 0).WithAuth("basic", cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	pipe, err := new(pipeline.Pipeline).WithJq(".values.power")
	if err != nil {
		return nil, err
	}
	power = power.WithPipeline(pipe)

	return NewConfigurable(power.FloatGetter())
}

func matchesIdentifier(id string, m discovergyMeter) bool {
	return id == m.MeterID || id == m.SerialNumber || id == m.FullSerialNumber
}
