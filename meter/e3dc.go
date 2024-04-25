package meter

import (
	"errors"
	"net"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spali/go-rscp/rscp"
	"github.com/spf13/cast"
)

type E3dc struct {
	capacity float64
	usage    templates.Usage // TODO check if we really want to depend on templates
	conn     *rscp.Client
}

func init() {
	registry.Add("e3dc-rscp", NewE3dcFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateE3dc -b *E3dc -r api.Meter -t "api.BatteryCapacity,Capacity,func() float64" -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error"

func NewE3dcFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Usage    templates.Usage
		Uri      string
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

	host, port_, err := net.SplitHostPort(util.DefaultPort(cc.Uri, 5033))
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(port_)

	cfg := rscp.ClientConfig{
		Address:           host,
		Port:              uint16(port),
		Username:          cc.User,
		Password:          cc.Password,
		Key:               cc.Key,
		ConnectionTimeout: cc.Timeout,
		SendTimeout:       cc.Timeout,
		ReceiveTimeout:    cc.Timeout,
	}

	return NewE3dc(cfg, cc.Usage, cc.Battery)
}

var e3dcOnce sync.Once

func NewE3dc(cfg rscp.ClientConfig, usage templates.Usage, batteryId uint16) (api.Meter, error) {
	e3dcOnce.Do(func() {
		log := util.NewLogger("e3dc")
		rscp.Log.SetLevel(logrus.DebugLevel)
		rscp.Log.SetOutput(log.TRACE.Writer())
	})

	conn, err := rscp.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	m := &E3dc{
		usage: usage,
		conn:  conn,
	}

	// decorate api.BatterySoc
	var (
		batterySoc      func() (float64, error)
		batteryCapacity func() float64
		batteryMode     func(api.BatteryMode) error
	)

	if usage == templates.UsageBattery {
		batterySoc = m.batterySoc
		batteryCapacity = m.batteryCapacity
		batteryMode = m.setBatteryMode

		res, err := m.conn.Send(rscp.Message{
			Tag:      rscp.BAT_REQ_DATA,
			DataType: rscp.Container,
			Value: []rscp.Message{
				{
					Tag:      rscp.BAT_INDEX,
					DataType: rscp.UInt16,
					Value:    batteryId,
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

		batSpec, err := rscpContains(res, rscp.BAT_SPECIFICATION)
		if err != nil {
			return nil, err
		}

		batCap, err := rscpContains(&batSpec, rscp.BAT_SPECIFIED_CAPACITY)
		if err != nil {
			return nil, err
		}

		cap, err := rscpValue(batCap, cast.ToFloat64E)
		if err != nil {
			return nil, err
		}

		m.capacity = cap / 1e3
	}

	return decorateE3dc(m, batteryCapacity, batterySoc, batteryMode), nil
}

func (m *E3dc) CurrentPower() (float64, error) {
	switch m.usage {
	case templates.UsageGrid:
		res, err := m.conn.Send(*rscp.NewMessage(rscp.EMS_REQ_POWER_GRID, nil))
		if err != nil {
			return 0, err
		}
		return rscpValue(*res, cast.ToFloat64E)

	case templates.UsagePV:
		res, err := m.conn.SendMultiple([]rscp.Message{
			*rscp.NewMessage(rscp.EMS_REQ_POWER_PV, nil),
			*rscp.NewMessage(rscp.EMS_REQ_POWER_ADD, nil),
		})
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
		pwr, err := rscpValue(*res, cast.ToFloat64E)
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
	return rscpValue(*res, cast.ToFloat64E)
}

func (m *E3dc) setBatteryMode(mode api.BatteryMode) error {
	var (
		res []rscp.Message
		err error
	)

	switch mode {
	case api.BatteryNormal:
		res, err = m.conn.SendMultiple([]rscp.Message{
			e3dcDischargeBatteryLimit(false, 0),
			e3dcBatteryCharge(0),
		})

	case api.BatteryHold:
		res, err = m.conn.SendMultiple([]rscp.Message{
			e3dcDischargeBatteryLimit(true, 0),
			e3dcBatteryCharge(0),
		})

	case api.BatteryCharge:
		res, err = m.conn.SendMultiple([]rscp.Message{
			e3dcDischargeBatteryLimit(true, 0),
			e3dcBatteryCharge(10000), // 10kWh
		})

	default:
		return api.ErrNotAvailable
	}

	if err == nil {
		err = rscpError(res...)
	}
	return err
}

func e3dcDischargeBatteryLimit(active bool, limit int) rscp.Message {
	contents := []rscp.Message{
		*rscp.NewMessage(rscp.EMS_POWER_LIMITS_USED, active),
	}

	if active {
		contents = append(contents, *rscp.NewMessage(rscp.EMS_MAX_DISCHARGE_POWER, limit))
	}

	return *rscp.NewMessage(rscp.EMS_REQ_SET_POWER_SETTINGS, contents)
}

func e3dcBatteryCharge(amount int) rscp.Message {
	return *rscp.NewMessage(rscp.EMS_REQ_START_MANUAL_CHARGE, amount)
}

func rscpError(msg ...rscp.Message) error {
	var errs []error
	for _, m := range msg {
		if m.DataType == rscp.Error {
			errs = append(errs, errors.New(rscp.RscpError(cast.ToUint32(m.Value)).String()))
		}
	}
	return errors.Join(errs...)
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

	res := slice[idx]
	return res, rscpError(res)
}

func rscpValue[T any](msg rscp.Message, fun func(any) (T, error)) (T, error) {
	var zero T
	if err := rscpError(msg); err != nil {
		return zero, err
	}

	return fun(msg.Value)
}

func rscpValues[T any](msg []rscp.Message, fun func(any) (T, error)) ([]T, error) {
	res := make([]T, 0, len(msg))

	for _, m := range msg {
		v, err := rscpValue(m, fun)
		if err != nil {
			return nil, err
		}

		res = append(res, v)
	}

	return res, nil
}
