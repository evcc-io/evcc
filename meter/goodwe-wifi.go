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

//go:generate go tool decorate -f decorateGoodWeWifi -b *goodWeWiFi -r api.Meter -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64"

// TODO deprecated remove

func NewGoodWeWifiFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		batteryCapacity `mapstructure:",squash"`
		URI, Usage      string
		Timeout         time.Duration
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewGoodWeWiFi(cc.URI, cc.Usage, cc.batteryCapacity.Decorator(), cc.Timeout)
}

func NewGoodWeWiFi(uri, usage string, capacity func() float64, timeout time.Duration) (api.Meter, error) {
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

	// decorate battery
	var (
		batterySoc      func() (float64, error)
		batteryCapacity func() float64
	)
	if usage == "battery" {
		batterySoc = res.batterySoc
		batteryCapacity = capacity
	}

	return decorateGoodWeWifi(res, batterySoc, batteryCapacity), nil
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
