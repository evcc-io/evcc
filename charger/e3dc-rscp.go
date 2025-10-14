package charger

import (
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/sirupsen/logrus"
	"github.com/spali/go-rscp/rscp"
	"github.com/spf13/cast"
)

type E3dc struct {
	mu          sync.Mutex
	deviceIdx   uint8 // device index, usually 0 for a single wallbox
	e3dcSunMode bool
	conn        *rscp.Client
	retry       func() error
}

func init() {
	registry.Add("e3dc-wallbox", NewE3dcFromConfig)
}

//go:generate go tool decorate -f decorateE3dc -b *E3dc -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)" -t "api.PhasePowers,Powers,func() (float64, float64, float64, error)"

func NewE3dcFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Uri         string
		User        string
		Password    string
		Key         string
		DeviceIndex uint8
		Timeout     time.Duration
		Phases1p3p  bool
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

	return NewE3dc(cfg, cc.DeviceIndex, cc.Phases1p3p)
}

var e3dcOnce sync.Once

func NewE3dc(cfg rscp.ClientConfig, deviceIdx uint8, phases bool) (api.Charger, error) {
	e3dcOnce.Do(func() {
		log := util.NewLogger("e3dc")
		rscp.Log.SetLevel(logrus.DebugLevel)
		rscp.Log.SetOutput(log.TRACE.Writer())
	})

	conn, err := rscp.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	wb := &E3dc{
		conn:        conn,
		deviceIdx:   deviceIdx,
		e3dcSunMode: false,
	}

	var phasesSet func(int) error
	var phasesGet func() (int, error)
	var phasePowers func() (float64, float64, float64, error)

	if phases && false {
		// Disabled Untested!!
		phasesSet = wb.setPhases1p3p
		phasesGet = wb.getPhases1p3p
	}
	phasePowers = wb.getPhasePowers

	wb.retry = func() (err error) {
		wb.conn.Disconnect()
		wb.conn, err = rscp.NewClient(cfg)
		return err
	}

	return decorateE3dc(wb, phasesSet, phasesGet, phasePowers), nil
}

// retryMessage executes a single message request with retry
func (wb *E3dc) retryMessage(msg rscp.Message) (*rscp.Message, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	result, err := wb.conn.Send(msg)
	if err == nil {
		return result, nil
	}

	if err := wb.retry(); err != nil {
		return nil, err
	}

	return wb.conn.Send(msg)
}

// retryMessages executes a multiple message request with retry
func (wb *E3dc) retryMessages(msgs []rscp.Message) ([]rscp.Message, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	result, err := wb.conn.SendMultiple(msgs)
	if err == nil {
		return result, nil
	}

	if err := wb.retry(); err != nil {
		return nil, err
	}

	return wb.conn.SendMultiple(msgs)
}

func (wb *E3dc) CurrentPower() (float64, error) {
	res1, res2, res3, err := wb.getPhasePowers()
	if err != nil {
		return 0, err
	}

	return res1 + res2 + res3, nil
}

func (wb *E3dc) getPhasePowers() (float64, float64, float64, error) {
	res, err := ReadComplexTags(wb, []rscp.Message{
		*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L1, nil),
		*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L2, nil),
		*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L3, nil)},
		[]rscp.Tag{rscp.WB_PM_POWER_L1, rscp.WB_PM_POWER_L1, rscp.WB_PM_POWER_L1}, cast.ToFloat64E)

	if err != nil {
		return 0, 0, 0, err
	}
	return res[0], res[1], res[2], nil // Power in W
}

func (wb *E3dc) Status() (api.ChargeStatus, error) {
	data, err := ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_ALG, nil)}, []rscp.Tag{rscp.WB_EXTERN_DATA}, cast.ToUint8SliceE)
	status_byte := data[0][2]
	if err != nil {
		return api.StatusNone, err
	}
	hasError := false
	if hasError {
		return api.StatusE, nil
	} else if (status_byte & 32) != 0 { // ChargingActive
		return api.StatusC, nil
	} else if (status_byte & 8) != 0 { // Car Plugged
		return api.StatusB, nil
	} else if (status_byte & 64) != 0 { // Error while charging
		return api.StatusB, nil
	}
	return api.StatusA, nil
}

// Enabled implements the api.Charger interface
func (wb *E3dc) Enabled() (bool, error) {
	stat, err := wb.Status()
	if err != nil {
		return false, err
	}

	switch stat {
	case api.StatusA:
		return false, nil
	case api.StatusB:
		return false, nil
	case api.StatusC:
		return true, nil
	case api.StatusE:
		return false, nil
	}

	res, err := ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_ABORT_CHARGING, nil)}, nil, cast.ToBoolE)
	if err != nil {
		return false, err
	}
	return !res[0], nil
}

// Enabled implements the api.Charger interface
func (wb *E3dc) Enable(enable bool) error {
	wb.GetEmsStatus()
	wb.SetEmsStatus(true, true, false)
	_, err := ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_SET_ABORT_CHARGING, !enable)}, nil, cast.ToUint8E)
	if err != nil {
		return err
	}

	// Set E3DC to Sunmode if charger is disabled to avoid Netcharging
	_, err = ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_SET_SUN_MODE_ACTIVE, !enable)}, nil, cast.ToUint8E)
	if err != nil {
		return err
	}
	// Set E3DC override power to 0 if charger is disabled to avoid charging
	if enable {
		_, err = wb.retryMessage(*rscp.NewMessage(rscp.EMS_REQ_SET_OVERRIDE_AVAILABLE_POWER, int32(11000)))
		if err != nil {
			return err
		}
	} else {
		_, err = wb.retryMessage(*rscp.NewMessage(rscp.EMS_REQ_SET_OVERRIDE_AVAILABLE_POWER, int32(0)))
		if err != nil {
			return err
		}
	}

	return nil
}

// Enabled implements the api.Charger interface
func (wb *E3dc) setPhases1p3p(phases int) error {
	switch phases {
	case 1:
		phases = 1
	case 2:
		phases = 3
	case 3:
		phases = 7
	}
	_, err := ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_SET_NUMBER_PHASES, uint8(phases))}, nil, cast.ToFloat64E)
	return err
}

func (wb *E3dc) getPhases1p3p() (int, error) {
	data, err := ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_PM_ACTIVE_PHASES, nil)}, nil, cast.ToUint8E)
	if err != nil {
		return 0, err
	}
	switch data[0] {
	case 1:
		return 1, nil
	case 3:
		return 2, nil
	case 7:
		return 3, nil
	default:
	}
	return 0, nil
}

// MaxCurrent implements the api.Charger interface
func (wb *E3dc) MaxCurrent(current int64) error {
	current_result, err := ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_SET_MAX_CHARGE_CURRENT, uint8(current))}, nil, cast.ToInt32E)
	if err != nil {
		return err
	}
	if current_result[0] != int32(current) {
		return errors.New("unable to set requested wallbox current")
	}

	return nil
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *E3dc) TotalEnergy() (float64, error) {
	data, err := ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_ENERGY_ALL, nil)}, nil, cast.ToFloat64E)
	if err != nil {
		return 0, err
	}
	return data[0] / 1000, nil // Wh to kWh
}

func (wb *E3dc) SetEmsStatus(e3dc_battery_prio bool, battery2car bool, epWbAllow bool) error {
	res, err := wb.retryMessages([]rscp.Message{
		*rscp.NewMessage(rscp.EMS_REQ_SET_BATTERY_BEFORE_CAR_MODE, e3dc_battery_prio),
		*rscp.NewMessage(rscp.EMS_REQ_SET_BATTERY_TO_CAR_MODE, battery2car),
		*rscp.NewMessage(rscp.EMS_REQ_SET_EP_WALLBOX_ALLOW, epWbAllow),
	})
	if err != nil {
		return err
	}
	tags, values, err := rscpValues(res, cast.ToUint8E)
	if err != nil {
		return err
	}
	for i, tag := range tags {
		switch tag {
		case rscp.EMS_SET_BATTERY_BEFORE_CAR_MODE:
			if values[i] == 0xFF {
				return errors.New("cannot set e3dc_battery_prio")
			}
		case rscp.EMS_SET_BATTERY_TO_CAR_MODE:
			if values[i] == 0xFF {
				return errors.New("cannot set battery2car")
			}
		case rscp.EMS_SET_EP_WALLBOX_ALLOW:
			if values[i] == 0xFF {
				return errors.New("cannot set epWbAllow")
			}
		}
	}
	return nil
}

func (wb *E3dc) GetEmsStatus() (bool, bool, bool, error) {
	var e3dc_battery_prio, battery_to_car, ems_allow_wb = false, false, false
	// ---------------------------------
	res, err := wb.retryMessages([]rscp.Message{
		*rscp.NewMessage(rscp.EMS_REQ_BATTERY_BEFORE_CAR_MODE, uint8(0)),
		*rscp.NewMessage(rscp.EMS_REQ_BATTERY_TO_CAR_MODE, uint8(0)),
		*rscp.NewMessage(rscp.EMS_REQ_GET_EP_WALLBOX_ALLOW, nil),
	})
	if err != nil {
		return false, false, false, err
	}
	tags, values, err := rscpValues(res, cast.ToUint8E)
	if err != nil {
		return false, false, false, err
	}
	for i, tag := range tags {
		switch tag {
		case rscp.EMS_BATTERY_BEFORE_CAR_MODE:
			e3dc_battery_prio = values[i] == 1
		case rscp.EMS_BATTERY_TO_CAR_MODE:
			battery_to_car = values[i] == 1
		case rscp.EMS_GET_EP_WALLBOX_ALLOW:
			ems_allow_wb = values[i] == 1
		}
	}

	return e3dc_battery_prio, battery_to_car, ems_allow_wb, nil
}

func ReadComplexTags[T any](wb *E3dc, req_msg []rscp.Message, rsp_tags []rscp.Tag, fun func(any) (T, error)) ([]T, error) {
	var zero []T
	var zeroval T

	req_msg = append(req_msg, *rscp.NewMessage(rscp.WB_INDEX, wb.deviceIdx))
	request := *rscp.NewMessage(rscp.WB_REQ_DATA, req_msg)

	res, err := wb.retryMessages([]rscp.Message{request})
	if err != nil {
		return zero, err
	}
	tags, values, err := rscpValuesWithTag(res, fun)
	if err != nil {
		return zero, err
	}
	if len(tags) == 1 && len(values) == 1 { // only WB_INDEX
		return []T{values[0]}, errors.New("Received only WB_INDEX, no data tag")
	} else if rsp_tags == nil && len(values) == 2 { // One overhead tag WB_INDEX
		for i, tag := range tags {
			if tag != rscp.WB_INDEX {
				return []T{values[i]}, nil
			}
		}
		return zero, nil
	} else if rsp_tags == nil {
		return zero, errors.New("unable to extract single value because received multiple values")
	} else {
		return_values := make([]T, len(rsp_tags))
		for i, tag := range rsp_tags {
			return_values[i] = zeroval
			for j, val := range values {
				if tag == tags[j] {
					return_values[i] = val
					break
				}
			}
		}
		return return_values, nil
	}
}

func rscpError(msg ...rscp.Message) error {
	var errs []error
	for _, m := range msg {
		if m.DataType == rscp.Error {
			errs = append(errs, errors.New(rscp.RscpError(cast.ToUint64(m.Value)).String()))
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

func rscpValues[T any](msg []rscp.Message, fun func(any) (T, error)) ([]rscp.Tag, []T, error) {
	vals := make([]T, 0, len(msg))
	tags := make([]rscp.Tag, 0, len(msg))

	for _, m := range msg {
		if m.DataType == rscp.Container {

		}
		v, err := rscpValue(m, fun)
		if err != nil {
			// Ignore on error, usually cannot cast to target
		}
		tags = append(tags, m.Tag)
		vals = append(vals, v)
	}

	return tags, vals, nil
}

func rscpValuesWithTag[T any](msg []rscp.Message, fun func(any) (T, error)) ([]rscp.Tag, []T, error) {

	var err error
	var tags []rscp.Tag
	var vals []T
	var tagss []rscp.Tag
	var valss []T
	var newVal T

	for _, m := range msg {
		for _, v_arr := range m.Value.([]rscp.Message) {
			switch v_arr.DataType {
			case rscp.Container:
				tags, vals, err = rscpValues(v_arr.Value.([]rscp.Message), fun)
			case rscp.Error:
				// Do nothing, just skip missing values e.g. when no external meter is configured
				vals = []T{}
			default:
				newVal, err = rscpValue(v_arr, fun)
				tags = []rscp.Tag{v_arr.Tag}
				vals = []T{newVal}
			}
			if err != nil {
				// ignore errors for now
			} else {
				tagss = append(tagss, tags...)
				valss = append(valss, vals...)
			}
		}
	}

	return tagss, valss, nil
}
