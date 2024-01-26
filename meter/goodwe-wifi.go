package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/goodwe"
	"github.com/evcc-io/evcc/util"
)

type goodWeWiFi struct {
	*goodwe.Server
	usage string
	uri   string
}

func init() {
	registry.Add("goodwe-wifi", NewGoodWeWifiFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateGoodWeWifi -b *goodWeWiFi -r api.Meter -t "api.Battery,Soc,func() (float64, error)"

func NewGoodWeWifiFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		capacity   `mapstructure:",squash"`
		URI, Usage string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewGoodWeWiFi(cc.URI, cc.Usage)
}

func NewGoodWeWiFi(uri string, usage string) (api.Meter, error) {
	instance, err := goodwe.Instance()
	if err != nil {
		return nil, err
	}

	res := &goodWeWiFi{
		Server: instance,
		usage:  usage,
		uri:    uri,
	}

	instance.AddInverter(uri)

	// decorate api.BatterySoc
	var batterySoc func() (float64, error)
	if usage == "battery" {
		batterySoc = res.batterySoc
	}

	return decorateGoodWeWifi(res, batterySoc), nil
}

func (m *goodWeWiFi) CurrentPower() (float64, error) {
	data := m.GetInverter(m.uri)

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
	return m.GetInverter(m.uri).Soc, nil
}
