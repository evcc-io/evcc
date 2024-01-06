package meter

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/discovergy"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
)

func init() {
	registry.Add("discovergy", NewDiscovergyFromConfig)
}

type Discovergy struct {
	dataG func() (discovergy.Reading, error)
	scale float64
}

// NewDiscovergyFromConfig creates a new configurable meter
func NewDiscovergyFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		User     string
		Password string
		Meter    string
		Scale    float64
		Cache    time.Duration
		Timeout  time.Duration
	}{
		Scale:   1,
		Cache:   time.Second,
		Timeout: time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	basicAuth := transport.BasicAuthHeader(cc.User, cc.Password)
	log := util.NewLogger("discgy").Redact(cc.User, cc.Password, cc.Meter, basicAuth)

	client := request.NewHelper(log)
	client.Transport = transport.BasicAuth(cc.User, cc.Password, client.Transport)

	var meters []discovergy.Meter
	if err := client.GetJSON(fmt.Sprintf("%s/meters", discovergy.API), &meters); err != nil {
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
		return nil, fmt.Errorf("could not determine meter id: %v", lo.Map(meters, func(m discovergy.Meter, _ int) string {
			return m.FullSerialNumber
		}))
	}

	dataG := provider.Cached(func() (discovergy.Reading, error) {
		var res discovergy.Reading
		uri := fmt.Sprintf("%s/last_reading?meterId=%s", discovergy.API, meterID)
		err := client.GetJSON(uri, &res)
		if err == nil && time.Since(time.UnixMilli(res.Time)) > cc.Timeout {
			err = api.ErrTimeout
		}
		return res, err
	}, cc.Cache)

	m := &Discovergy{
		dataG: dataG,
		scale: cc.Scale,
	}

	return m, nil
}

func matchesIdentifier(id string, m discovergy.Meter) bool {
	return id == m.MeterID || id == m.SerialNumber || id == m.FullSerialNumber
}

func (m *Discovergy) CurrentPower() (float64, error) {
	res, err := m.dataG()
	return m.scale * float64(res.Values.Power) / 1e3, err
}

var _ api.MeterEnergy = (*Discovergy)(nil)

func (m *Discovergy) TotalEnergy() (float64, error) {
	res, err := m.dataG()
	return float64(res.Values.Energy) / 1e10, err
}
