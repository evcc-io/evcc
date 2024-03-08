package meter

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/goodwe"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type goodWeWiFi struct {
	usage    string
	inverter *util.Monitor[goodwe.Inverter]
}

func init() {
	registry.Add("goodwe-wifi", NewGoodWeWifiFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateGoodWeWifi -b *goodWeWiFi -r api.Meter -t "api.Battery,Soc,func() (float64, error)"

func NewGoodWeWifiFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		capacity   `mapstructure:",squash"`
		URI, Usage string
		Timeout    time.Duration
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewGoodWeWiFi(cc.URI, cc.Usage, cc.Timeout)
}

func NewGoodWeWiFi(uri, usage string, timeout time.Duration) (api.Meter, error) {
	instance, err := goodwe.Instance(util.NewLogger("goodwe-wifi"))
	if err != nil {
		return nil, err
	}

	inverter := instance.GetInverter(uri)
	if inverter == nil {
		inverter = instance.AddInverter(uri, timeout)
	}

	res := &goodWeWiFi{
		usage:    usage,
		inverter: inverter,
	}

	// decorate api.BatterySoc
	var batterySoc func() (float64, error)
	if usage == "battery" {
		batterySoc = res.batterySoc
	}

	return decorateGoodWeWifi(res, batterySoc), nil
}

func (m *goodWeWiFi) CurrentPower() (float64, error) {
	data, err := m.inverter.Get()
	if err != nil {
		return 0, err
	}

	switch m.usage {
	case "grid":
		return data.NetPower, nil
	case "pv":
		return data.PvPower, nil
	case "battery":
		return data.BatteryPower, nil
	}
	return 0, api.ErrNotAvailable
}

func (m *goodWeWiFi) batterySoc() (float64, error) {
	data, err := m.inverter.Get()
	return data.Soc, err
}
