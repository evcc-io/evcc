package meter

import (
	"errors"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/samber/lo"
	"github.com/spali/go-rscp/rscp"
	"github.com/spf13/cast"
)

type E3dc struct {
	capacity float64
	usage    templates.Usage // TODO check if we really want to depend on templates
	conn     *rscp.Client
}

func init() {
	registry.Add("e3dc-2", NewE3dcFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateE3dc -b *E3dc -r api.Meter -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64"

func NewE3dcFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Usage    templates.Usage
		Host     string
		Port     uint16
		User     string
		Password string
		Key      string
		Battery  uint16 // battery id
		Timeout  time.Duration
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	cfg := rscp.ClientConfig{
		Address:           cc.Host,
		Port:              cc.Port,
		Username:          cc.User,
		Password:          cc.Password,
		Key:               cc.Key,
		ConnectionTimeout: cc.Timeout,
		SendTimeout:       cc.Timeout,
		ReceiveTimeout:    cc.Timeout,
	}

	return NewE3dc(cfg, cc.Usage, cc.Battery)
}

func NewE3dc(cfg rscp.ClientConfig, usage templates.Usage, batteryId uint16) (api.Meter, error) {
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
	var (
		batterySoc      func() (float64, error)
		batteryCapacity func() float64
	)

	if usage == templates.UsageBattery {
		batterySoc = res.batterySoc
		batteryCapacity = res.batteryCapacity

		resp, err := res.conn.Send(rscp.Message{
			Tag:      rscp.BAT_REQ_DATA,
			DataType: rscp.Container,
			Value: []rscp.Message{
				{
					Tag:      rscp.BAT_INDEX,
					DataType: rscp.UInt16,
					Value:    uint16(batteryId),
				},
				{
					Tag:      rscp.BAT_REQ_SPECIFICATION,
					DataType: rscp.None,
				},
			},
		})
		if err != nil {
			return nil, err
		}

		batSpec, err := rscpContains(resp, rscp.BAT_SPECIFICATION)
		if err != nil {
			return nil, err
		}

		batCap, err := rscpContains(&batSpec, rscp.BAT_SPECIFIED_CAPACITY)
		if err != nil {
			return nil, err
		}

		cap, err := cast.ToFloat64E(batCap.Value)
		if err != nil {
			return nil, err
		}

		res.capacity = cap / 1e3
	}

	return decorateE3dc(res, batterySoc, batteryCapacity), nil
}

func rscpContains(msg *rscp.Message, tag rscp.Tag) (rscp.Message, error) {
	var zero rscp.Message

	slice, ok := msg.Value.([]rscp.Message)
	if !ok {
		return zero, errors.New("not a slice looking for " + tag.String())
	}

	idx := slices.IndexFunc(slice, func(m rscp.Message) bool {
		return m.Tag == tag
	})
	if idx < 0 {
		return zero, errors.New("missing " + tag.String())
	}

	return slice[idx], nil
}

func rscpValues[T any](msg []rscp.Message, fun func(any) (T, error)) ([]T, error) {
	res := make([]T, 0, len(msg))
	for _, m := range msg {
		v, err := fun(m.Value)
		if err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return res, nil
}

func (m *E3dc) CurrentPower() (float64, error) {
	switch m.usage {
	case templates.UsageGrid:
		res, err := m.conn.Send(*rscp.NewMessage(rscp.EMS_REQ_POWER_GRID, nil))
		if err != nil {
			return 0, err
		}
		return cast.ToFloat64E(res.Value)

	case templates.UsagePV:
		res, err := m.conn.SendMultiple([]rscp.Message{*rscp.NewMessage(rscp.EMS_REQ_POWER_PV, nil), *rscp.NewMessage(rscp.EMS_REQ_POWER_ADD, nil)})
		if err != nil {
			return 0, err
		}

		values, err := rscpValues(res, cast.ToFloat64E)
		if err != nil {
			return 0, err
		}

		return lo.Sum(values), nil

	case templates.UsageBattery:
		res, err := m.conn.Send(*rscp.NewMessage(rscp.EMS_REQ_POWER_BAT, nil))
		if err != nil {
			return 0, err
		}
		pwr, err := cast.ToFloat64E(res.Value)
		if err != nil {
			return 0, err
		}

		return -pwr, nil

	default:
		return 0, api.ErrNotAvailable
	}
}

func (m *E3dc) batteryCapacity() float64 {
	return m.capacity
}

func (m *E3dc) batterySoc() (float64, error) {
	res, err := m.conn.Send(*rscp.NewMessage(rscp.EMS_REQ_BAT_SOC, nil))
	if err != nil {
		return 0, err
	}
	return cast.ToFloat64E(res.Value)
}
