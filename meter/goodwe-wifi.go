package meter

import (
	"encoding/binary"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	gridx "github.com/grid-x/modbus"
)

type goodWeWiFi struct {
	log   *util.Logger
	usage string
	conn  gridx.Client
}

const (
	goodwePv1Power    = 0x8921
	goodwePv2Power    = 0x8925
	goodweActivePower = 0x8944
	goodweBatterySoc  = 0x908F
)

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
	handler := gridx.NewRTUOverUDPClientHandler(uri)
	conn := gridx.NewClient(handler)

	res := &goodWeWiFi{
		log:   util.NewLogger("goodwe-wifi"),
		usage: usage,
		conn:  conn,
	}

	handler.Logger = res

	// decorate api.BatterySoc
	var batterySoc func() (float64, error)
	if usage == "battery" {
		batterySoc = res.batterySoc
	}

	return decorateGoodWeWifi(res, batterySoc), nil
}

func (m *goodWeWiFi) Printf(format string, v ...interface{}) {
	// TODO modbus format
	m.log.TRACE.Printf(format, v...)
}

func (m *goodWeWiFi) CurrentPower() (float64, error) {
	switch m.usage {
	case "grid":
		b, err := m.conn.ReadInputRegisters(goodweActivePower, 1)
		if err != nil {
			return 0, err
		}
		return float64(int16(binary.BigEndian.Uint16(b))), nil

	case "pv":
		b, err := m.conn.ReadInputRegisters(goodwePv1Power, 2)
		if err != nil {
			return 0, err
		}
		return float64(binary.BigEndian.Uint32(b)), nil

	case "battery":
		return 0, api.ErrNotAvailable
	}

	return 0, api.ErrNotAvailable
}

func (m *goodWeWiFi) batterySoc() (float64, error) {
	b, err := m.conn.ReadInputRegisters(goodweBatterySoc, 1)
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint16(b)), nil
}
