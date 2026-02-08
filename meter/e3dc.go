package meter

import (
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/sirupsen/logrus"
	"github.com/spali/go-rscp/rscp"
	"github.com/spf13/cast"
)

type E3dc struct {
	mu             sync.Mutex
	dischargeLimit uint32
	externalPower  bool            // whether to include power of external sources
	usage          templates.Usage // TODO check if we really want to depend on templates
	conn           *rscp.Client
	retry          func() error
}

func init() {
	registry.Add("e3dc-rscp", NewE3dcFromConfig)
}

//go:generate go tool decorate -f decorateE3dc -b *E3dc -r api.Meter -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error" -t "api.MaxACPowerGetter,MaxACPower,func() float64"

func NewE3dcFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		batteryCapacity `mapstructure:",squash"`
		pvMaxACPower    `mapstructure:",squash"`
		Usage           templates.Usage
		Uri             string
		User            string
		Password        string
		Key             string
		ExternalPower   bool
		DischargeLimit  uint32
		Timeout         time.Duration
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

	return NewE3dc(cfg, cc.Usage, cc.DischargeLimit, cc.ExternalPower, cc.batteryCapacity.Decorator(), cc.pvMaxACPower.Decorator())
}

var e3dcOnce sync.Once

func NewE3dc(cfg rscp.ClientConfig, usage templates.Usage, dischargeLimit uint32, externalPower bool, capacity, maxacpower func() float64) (api.Meter, error) {
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
		usage:          usage,
		conn:           conn,
		externalPower:  externalPower,
		dischargeLimit: dischargeLimit,
	}

	m.retry = func() (err error) {
		m.conn.Disconnect()
		m.conn, err = rscp.NewClient(cfg)
		return err
	}

	// decorate battery
	var (
		batteryCapacity func() float64
		batterySoc      func() (float64, error)
		batteryMode     func(api.BatteryMode) error
	)

	if usage == templates.UsageBattery {
		batteryCapacity = capacity
		batterySoc = m.batterySoc
		batteryMode = m.setBatteryMode
	}

	return decorateE3dc(m, batterySoc, batteryCapacity, batteryMode, maxacpower), nil
}

// retryMessage executes a single message request with retry
func (m *E3dc) retryMessage(msg rscp.Message) (*rscp.Message, error) {
	result, err := m.conn.Send(msg)
	if err == nil {
		return result, nil
	}

	if err := m.retry(); err != nil {
		return nil, err
	}

	return m.conn.Send(msg)
}

// retryMessages executes a multiple message request with retry
func (m *E3dc) retryMessages(msgs []rscp.Message) ([]rscp.Message, error) {
	result, err := m.conn.SendMultiple(msgs)
	if err == nil {
		return result, nil
	}

	if err := m.retry(); err != nil {
		return nil, err
	}

	return m.conn.SendMultiple(msgs)
}

func (m *E3dc) CurrentPower() (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch m.usage {
	case templates.UsageGrid:
		res, err := m.retryMessage(*rscp.NewMessage(rscp.EMS_REQ_POWER_GRID, nil))
		if err != nil {
			return 0, err
		}
		return rscpValue(*res, cast.ToFloat64E)

	case templates.UsagePV:
		res, err := m.retryMessages([]rscp.Message{
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

		if m.externalPower {
			return values[0] - values[1], nil
		}

		return values[0], nil

	case templates.UsageBattery:
		res, err := m.retryMessage(*rscp.NewMessage(rscp.EMS_REQ_POWER_BAT, nil))
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

func (m *E3dc) batterySoc() (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res, err := m.retryMessage(*rscp.NewMessage(rscp.EMS_REQ_BAT_SOC, nil))
	if err != nil {
		return 0, err
	}

	return rscpValue(*res, cast.ToFloat64E)
}

func (m *E3dc) setBatteryMode(mode api.BatteryMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var messages []rscp.Message
	switch mode {
	case api.BatteryNormal:
		messages = []rscp.Message{
			e3dcDischargeBatteryLimit(false, 0),
			e3dcBatteryCharge(0),
		}
	case api.BatteryHold:
		messages = []rscp.Message{
			e3dcDischargeBatteryLimit(true, m.dischargeLimit),
			e3dcBatteryCharge(0),
		}
	case api.BatteryCharge:
		messages = []rscp.Message{
			e3dcDischargeBatteryLimit(false, 0),
			e3dcBatteryCharge(50000), // max. 50kWh
		}
	default:
		return api.ErrNotAvailable
	}

	res, err := m.retryMessages(messages)
	if err != nil {
		return err
	}

	return rscpError(res...)
}

func e3dcDischargeBatteryLimit(active bool, limit uint32) rscp.Message {
	contents := []rscp.Message{
		*rscp.NewMessage(rscp.EMS_POWER_LIMITS_USED, active),
	}

	if active {
		contents = append(contents, *rscp.NewMessage(rscp.EMS_MAX_DISCHARGE_POWER, limit))
	}

	return *rscp.NewMessage(rscp.EMS_REQ_SET_POWER_SETTINGS, contents)
}

func e3dcBatteryCharge(amount uint32) rscp.Message {
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
