package meter

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/spali/go-rscp/rscp"
	"github.com/spf13/cast"
)

type E3dc struct {
	// TODO check if we really want to depend on templates
	usage templates.Usage
	conn  *rscp.Client
}

func init() {
	registry.Add("e3dc-2", NewE3dcFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateE3dc -b *E3dc -r api.Meter -t "api.Battery,Soc,func() (float64, error)"

func NewE3dcFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		capacity `mapstructure:",squash"`
		Usage    templates.Usage
		Address  string
		Port     uint16
		Username string
		Password string
		Key      string
		Timeout  time.Duration
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	cfg := rscp.ClientConfig{
		Address:           cc.Address,
		Port:              cc.Port,
		Username:          cc.Username,
		Password:          cc.Password,
		Key:               cc.Key,
		ConnectionTimeout: cc.Timeout,
		SendTimeout:       cc.Timeout,
		ReceiveTimeout:    cc.Timeout,
	}

	return NewE3dc(cc.Usage, cfg)
}

func NewE3dc(usage templates.Usage, cfg rscp.ClientConfig) (api.Meter, error) {
	// util.NewLogger("e3dc")
	conn, err := rscp.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	res := &E3dc{
		usage: usage,
		conn:  conn,
	}

	// decorate api.BatterySoc
	var batterySoc func() (float64, error)
	if usage == templates.UsageBattery {
		batterySoc = res.batterySoc
	}

	return decorateE3dc(res, batterySoc), nil
}

func (m *E3dc) CurrentPower() (float64, error) {
	var tag rscp.Tag

	switch m.usage {
	case templates.UsageGrid:
		tag = rscp.EMS_REQ_POWER_GRID
	case templates.UsagePV:
		tag = rscp.EMS_POWER_PV
	case templates.UsageBattery:
		tag = rscp.EMS_POWER_BAT
	default:
		return 0, api.ErrNotAvailable
	}

	res, err := m.conn.Send(*rscp.NewMessage(tag, nil))
	if err != nil {
		return 0, err
	}

	return cast.ToFloat64E(res.Value)
}

func (m *E3dc) batterySoc() (float64, error) {
	res, err := m.conn.Send(*rscp.NewMessage(rscp.EMS_REQ_BAT_SOC, nil))
	if err != nil {
		return 0, err
	}
	return cast.ToFloat64E(res.Value)
}
