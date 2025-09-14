// queries PM #1 and PVI #1 only

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
	usage          templates.Usage // TODO check if we really want to depend on templates
	conn           *rscp.Client
	retry          func() error
}

func init() {
	registry.Add("e3dc-rscp", NewE3dcFromConfig)
}

//go:generate go tool decorate -f decorateE3dc -b *E3dc -r api.Meter -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error" -t "api.MaxACPowerGetter,MaxACPower,func() float64"

func NewE3dcFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		batteryCapacity `mapstructure:",squash"`
		pvMaxACPower    `mapstructure:",squash"`
		Usage           templates.Usage
		Uri             string
		User            string
		Password        string
		Key             string
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

	return NewE3dc(cfg, cc.Usage, cc.DischargeLimit, cc.batteryCapacity.Decorator(), cc.pvMaxACPower.Decorator())
}

var e3dcOnce sync.Once

func NewE3dc(cfg rscp.ClientConfig, usage templates.Usage, dischargeLimit uint32, capacity, maxacpower func() float64) (api.Meter, error) {
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

func extractValueByTag[T any](msg rscp.Message, wantedTag rscp.Tag, fun func(any) (T, error)) (T, bool, error) {
	var zero T

	if msg.DataType != rscp.Container {
		if msg.Tag == wantedTag {
			v, err := rscpValue(msg, fun)
			if err != nil {
				return zero, false, err
			}
			return v, true, nil
		}
	} else {
		if nestedMessage, ok := msg.Value.([]rscp.Message); ok {
			for _, m := range nestedMessage {
				// ok == tag found
				if val, ok, err := extractValueByTag(m, wantedTag, fun); ok {
					return val, ok, err
				}
			}
		} else {
			return zero, false, nil
		}
	}

	return zero, false, nil
}

var _ api.Meter = (*E3dc)(nil)

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

		return values[0] - values[1], nil

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

var _ api.PhaseVoltages = (*E3dc)(nil)

func (m *E3dc) Voltages() (float64, float64, float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch m.usage {
	case templates.UsageGrid:
		res, err := m.retryMessage(rscp.Message{
			Tag:      rscp.PM_REQ_DATA,
			DataType: rscp.Container,
			Value: []rscp.Message{
				{
					Tag:      rscp.PM_INDEX,
					DataType: rscp.UInt16,
					Value:    uint16(0), // PM #1
				},
				{
					Tag:      rscp.PM_REQ_VOLTAGE_L1,
					DataType: rscp.None,
					// Value:    nil,
				},
				{
					Tag:      rscp.PM_REQ_VOLTAGE_L2,
					DataType: rscp.None,
					// Value:    nil,
				},
				{
					Tag:      rscp.PM_REQ_VOLTAGE_L3,
					DataType: rscp.None,
					// Value:    nil,
				},
			},
		})

		if err != nil {
			return 0, 0, 0, err
		}

		voltageL1, found, err := extractValueByTag(*res, rscp.PM_VOLTAGE_L1, cast.ToFloat64E)
		if !found || err != nil {
			return 0, 0, 0, err
		}
		voltageL2, found, err := extractValueByTag(*res, rscp.PM_VOLTAGE_L2, cast.ToFloat64E)
		if !found || err != nil {
			return 0, 0, 0, err
		}
		voltageL3, found, err := extractValueByTag(*res, rscp.PM_VOLTAGE_L3, cast.ToFloat64E)
		if !found || err != nil {
			return 0, 0, 0, err
		}
		return voltageL1, voltageL2, voltageL3, nil

	default:
		return 0, 0, 0, api.ErrNotAvailable
	}
}

var _ api.PhaseCurrents = (*E3dc)(nil)

func (m *E3dc) Currents() (float64, float64, float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch m.usage {
	case templates.UsageGrid:
		res, err := m.retryMessage(rscp.Message{
			Tag:      rscp.PM_REQ_DATA,
			DataType: rscp.Container,
			Value: []rscp.Message{
				{
					Tag:      rscp.PM_INDEX,
					DataType: rscp.UInt16,
					Value:    uint16(0), // PM #1
				},
				{
					Tag:      rscp.PM_REQ_POWER_L1,
					DataType: rscp.None,
					Value:    nil,
				},
				{
					Tag:      rscp.PM_REQ_POWER_L2,
					DataType: rscp.None,
					Value:    nil,
				},
				{
					Tag:      rscp.PM_REQ_POWER_L3,
					DataType: rscp.None,
					Value:    nil,
				},
				{
					Tag:      rscp.PM_REQ_VOLTAGE_L1,
					DataType: rscp.None,
					Value:    nil,
				},
				{
					Tag:      rscp.PM_REQ_VOLTAGE_L2,
					DataType: rscp.None,
					Value:    nil,
				},
				{
					Tag:      rscp.PM_REQ_VOLTAGE_L3,
					DataType: rscp.None,
					Value:    nil,
				},
			},
		})

		if err != nil {
			return 0, 0, 0, err
		}

		powerL1, found, err := extractValueByTag(*res, rscp.PM_POWER_L1, cast.ToFloat64E)
		if !found || err != nil {
			return 0, 0, 0, err
		}
		powerL2, found, err := extractValueByTag(*res, rscp.PM_POWER_L2, cast.ToFloat64E)
		if !found || err != nil {
			return 0, 0, 0, err
		}
		powerL3, found, err := extractValueByTag(*res, rscp.PM_POWER_L3, cast.ToFloat64E)
		if !found || err != nil {
			return 0, 0, 0, err
		}
		voltageL1, found, err := extractValueByTag(*res, rscp.PM_VOLTAGE_L1, cast.ToFloat64E)
		if !found || err != nil {
			return 0, 0, 0, err
		}
		voltageL2, found, err := extractValueByTag(*res, rscp.PM_VOLTAGE_L2, cast.ToFloat64E)
		if !found || err != nil {
			return 0, 0, 0, err
		}
		voltageL3, found, err := extractValueByTag(*res, rscp.PM_VOLTAGE_L3, cast.ToFloat64E)
		if !found || err != nil {
			return 0, 0, 0, err
		}
		var currentL1, currentL2, currentL3 float64
		if voltageL1 != 0 {
			currentL1 = powerL1 / voltageL1
		} else {
			return 0, 0, 0, nil
		}
		if voltageL2 != 0 {
			currentL2 = powerL2 / voltageL2
		} else {
			return 0, 0, 0, nil
		}
		if voltageL3 != 0 {
			currentL3 = powerL3 / voltageL3
		} else {
			return 0, 0, 0, nil
		}

		return currentL1, currentL2, currentL3, nil

	default:
		return 0, 0, 0, api.ErrNotAvailable
	}
}

var _ api.MeterEnergy = (*E3dc)(nil)

func (m *E3dc) TotalEnergy() (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var energyPerPhase [3]float64

	switch m.usage {
	case templates.UsageGrid:
		return 0, api.ErrNotAvailable

	case templates.UsagePV:
		for phase := range 3 {
			res, err := m.retryMessage(rscp.Message{
				Tag:      rscp.PVI_REQ_DATA,
				DataType: rscp.Container,
				Value: []rscp.Message{
					{
						Tag:      rscp.PVI_INDEX,
						DataType: rscp.UInt16,
						Value:    uint16(0), // PVI #1 = 0
					},
					{
						Tag:      rscp.PVI_REQ_AC_ENERGY_ALL,
						DataType: rscp.UInt16,
						Value:    uint16(phase), // phase
					},
				},
			})
			if err != nil {
				return 0, err
			}

			val, found, err := extractValueByTag(*res, rscp.PVI_VALUE, cast.ToFloat64E)
			if !found {
				return 0, err
			}
			energyPerPhase[phase] = val
		}

		return (energyPerPhase[0] + energyPerPhase[1] + energyPerPhase[2]) / 1000, nil // Wh -> kWh

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
