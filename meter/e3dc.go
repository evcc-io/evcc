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

//go:generate go tool decorate -f decorateE3dc -b *E3dc -r api.Meter -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error" -t "api.MaxACPowerGetter,MaxACPower,func() float64" -t "api.PhaseVoltages,Voltages,func() (float64,float64,float64, error)" -t "api.PhaseCurrents,Currents,func() (float64,float64,float64, error)" -t "api.PhasePowers,Powers,func() (float64,float64,float64, error)"

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

	// decorate battery // grid
	var (
		batteryCapacity func() float64
		batterySoc      func() (float64, error)
		batteryMode     func(api.BatteryMode) error
		phaseCurrents   func() (float64, float64, float64, error)
		phaseVoltages   func() (float64, float64, float64, error)
		phasePower      func() (float64, float64, float64, error)
	)

	switch usage {
	case templates.UsageBattery:
		batteryCapacity = capacity
		batterySoc = m.batterySoc
		batteryMode = m.setBatteryMode
	case templates.UsageGrid:
		phaseVoltages = m.getPhaseVoltages
		phaseCurrents = m.getPhaseCurrents
		phasePower = m.getPhasePower
	}

	return decorateE3dc(m, batterySoc, batteryCapacity, batteryMode, maxacpower, phaseVoltages, phaseCurrents, phasePower), nil
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

func (m *E3dc) getPhaseVoltages() (float64, float64, float64, error) {
	switch m.usage {
	case templates.UsageGrid:
		voltages, _, _, _, err := m.ReadFromPM(0, 1)
		if err != nil {
			return 0, 0, 0, err
		}
		return voltages[0], voltages[1], voltages[2], nil
	}
	return 0, 0, 0, nil
}

func (m *E3dc) getPhaseCurrents() (float64, float64, float64, error) {

	switch m.usage {
	case templates.UsageGrid:
		_, currents, _, _, err := m.ReadFromPM(0, 1) // Verify type 1 for root meter
		if err != nil {
			return 0, 0, 0, err
		}
		return currents[0], currents[1], currents[2], nil
	}
	return 0, 0, 0, nil
}

func (m *E3dc) getPhasePower() (float64, float64, float64, error) {

	switch m.usage {
	case templates.UsageGrid:
		_, _, power, _, err := m.ReadFromPM(0, 1) // Verify type 1 for root meter
		if err != nil {
			return 0, 0, 0, err
		}
		return power[0], power[1], power[2], nil
	}
	return 0, 0, 0, nil
}

// Read voltage, current, power and energy from Powermeter namespace for the given powermeter index (usually 0 = ROOT = Grid and 1 = external PV inverter/generator)
func (m *E3dc) ReadFromPM(pm_idx uint16, verfy_type int16) ([3]float64, [3]float64, [3]float64, float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var voltage = [3]float64{0}
	var power = [3]float64{0}
	var current = [3]float64{0}
	var total_energy = float64(0)

	request := *rscp.NewMessage(
		rscp.PM_REQ_DATA, []rscp.Message{
			*rscp.NewMessage(rscp.PM_INDEX, pm_idx),
			*rscp.NewMessage(rscp.PM_REQ_TYPE, nil),
			*rscp.NewMessage(rscp.PM_REQ_POWER_L1, nil),
			*rscp.NewMessage(rscp.PM_REQ_POWER_L2, nil),
			*rscp.NewMessage(rscp.PM_REQ_POWER_L3, nil),
			*rscp.NewMessage(rscp.PM_REQ_ENERGY_L1, nil),
			*rscp.NewMessage(rscp.PM_REQ_ENERGY_L2, nil),
			*rscp.NewMessage(rscp.PM_REQ_ENERGY_L3, nil),
			*rscp.NewMessage(rscp.PM_REQ_VOLTAGE_L1, nil),
			*rscp.NewMessage(rscp.PM_REQ_VOLTAGE_L2, nil),
			*rscp.NewMessage(rscp.PM_REQ_VOLTAGE_L3, nil),
		})
	res, err := m.retryMessages([]rscp.Message{request})
	if err != nil {
		return [3]float64{0}, [3]float64{0}, [3]float64{0}, 0, err
	}

	tags, values, err := rscpValuesWithTag(res, cast.ToFloat64E)
	if err != nil {
		return [3]float64{0}, [3]float64{0}, [3]float64{0}, 0, err
	}

	for tag_idx := range tags {
		switch tags[tag_idx] {
		case rscp.PM_TYPE:
			if verfy_type != -1 && values[tag_idx][0] != float64(verfy_type) {
				return [3]float64{0}, [3]float64{0}, [3]float64{0}, 0, nil // Not the requested meter type
			}
		case rscp.PM_VOLTAGE_L1:
			voltage[0] = values[tag_idx][0]
			voltage[1] = values[tag_idx][0]
		case rscp.PM_VOLTAGE_L2:
			//voltage[1] = values[tag_idx][0]  // False reading on L2 use L1 instead
		case rscp.PM_VOLTAGE_L3:
			voltage[2] = values[tag_idx][0]
		case rscp.PM_POWER_L1:
			power[0] = values[tag_idx][0]
		case rscp.PM_POWER_L2:
			power[1] = values[tag_idx][0]
		case rscp.PM_POWER_L3:
			power[2] = values[tag_idx][0]
		case rscp.PM_ENERGY_L1:
			total_energy += values[tag_idx][0]
		case rscp.PM_ENERGY_L2:
			total_energy += values[tag_idx][0]
		case rscp.PM_ENERGY_L3:
			total_energy += values[tag_idx][0]
		default:
		}
	}
	if voltage[0] > 100 && voltage[1] > 100 && voltage[2] > 100 {
		current[0] = power[0] / voltage[0]
		current[1] = power[1] / voltage[1]
		current[2] = power[2] / voltage[2]
	}
	return voltage, current, power, total_energy / 1000, nil
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

func rscpValuesWithTag[T any](msg []rscp.Message, fun func(any) (T, error)) ([]rscp.Tag, [][]T, error) {
	//res := make([]T, 0, len(msg))
	res := [][]T{}
	var tags []rscp.Tag
	var err error
	var v []T
	var newVal T
	for _, m := range msg {
		for _, v_arr := range m.Value.([]rscp.Message) {
			if v_arr.DataType == rscp.Container {
				v, err = rscpValues(v_arr.Value.([]rscp.Message), fun)
			} else {
				newVal, err = rscpValue(v_arr, fun)
				v = []T{newVal}
			}
			if err != nil {
				return nil, nil, err
			}
			tags = append(tags, v_arr.Tag)
			res = append(res, v)
		}
	}

	return tags, res, nil
}
