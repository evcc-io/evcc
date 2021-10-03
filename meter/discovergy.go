package meter

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/basicauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/thoas/go-funk"
)

const discovergyAPI = "https://api.discovergy.com/public/v1"

type Discovergy struct {
	*request.Helper
}

func init() {
	registry.Add("discovergy", NewDiscovergyFromConfig)
}

type discovergyMeter struct {
	MeterID          string `json:"meterId"`
	SerialNumber     string `json:"serialNumber"`
	FullSerialNumber string `json:"fullSerialNumber"`
}

type discovergyLastReading struct {
	Values struct {
		Power float64 `json:"power"`
	} `json:"values"`
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

	client := request.NewHelper(util.NewLogger("discgy"))

	c := &Discovergy{
		Helper: client,
	}

	c.Client.Transport = basicauth.NewTransport(cc.User, cc.Password, c.Client.Transport)

	var meters []discovergyMeter
	if err := c.GetJSON(fmt.Sprintf("%s/meters", discovergyAPI), &meters); err != nil {
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

	var lastreading discovergyLastReading
	if err := c.GetJSON(fmt.Sprintf("%s/last_reading?meterId=%s", discovergyAPI, meterID), &lastreading); err != nil {
		return nil, err
	}

	// return lastreading.Values.Power, nil

	// return NewConfigurable(power.FloatGetter()), nil)
}

func matchesIdentifier(id string, m discovergyMeter) bool {
	return id == m.MeterID || id == m.SerialNumber || id == m.FullSerialNumber
}
