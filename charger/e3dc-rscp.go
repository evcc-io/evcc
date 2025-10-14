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
	deviceIdx   uint8 // device index, usually 0 needed for powermeter and wallbox
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
	wb.TestWallboxExternData()
	wb.GetEmsStatus()
	wb.TestWallboxTags()
	//
	res22, err22 := wb.retryMessage(*rscp.NewMessage(rscp.EMS_REQ_SET_OVERRIDE_AVAILABLE_POWER, int32(0)))
	if err22 != nil {
		return false, err22
	}
	_, err33 := rscpValue(*res22, cast.ToBoolE)
	if err33 != nil {
		return false, err33
	}

	stat, err4 := wb.Status()
	if err4 != nil {
		return false, err4
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
	_, err = ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_SET_SUN_MODE_ACTIVE, true)}, nil, cast.ToUint8E)
	if err != nil {
		return err
	}
	// Set E3DC override power to 0 if charger is disabled to avoid charging
	if enable {
		_, err = ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.EMS_REQ_SET_OVERRIDE_AVAILABLE_POWER, uint32(11000))}, nil, cast.ToUint8E)
		if err != nil {
			return err
		}
	} else {
		_, err = ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.EMS_REQ_SET_OVERRIDE_AVAILABLE_POWER, uint32(0))}, nil, cast.ToUint8E)
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

	// ---------------------------------
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
	wb.mu.Lock()
	defer wb.mu.Unlock()
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
	if rsp_tags == nil && len(values) == 2 { // One overhead tag WB_INDEX
		for i, tag := range tags {
			if tag != rscp.WB_INDEX {
				return []T{values[i]}, nil
				break
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

// ######################################################################################################
// Test functions and tags
func (m *E3dc) ReadFromWBTestTags(wb_idx uint8) ([3]float64, float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var power = [3]float64{0}
	var total_energy = float64(0)

	request := *rscp.NewMessage(
		rscp.WB_REQ_DATA, []rscp.Message{
			*rscp.NewMessage(rscp.WB_INDEX, wb_idx),
			*rscp.NewMessage(rscp.WB_REQ_ENERGY_ALL, nil),         // Uint32 Energie Gesamt in Wh
			*rscp.NewMessage(rscp.WB_REQ_ENERGY_SOLAR, nil),       // Uint32 Energie Solar in Wh
			*rscp.NewMessage(rscp.WB_REQ_SOC, nil),                // Uint16 SOC in %
			*rscp.NewMessage(rscp.WB_REQ_ERROR_CODE, nil),         // Uint8
			*rscp.NewMessage(rscp.WB_REQ_DEVICE_NAME, nil),        // String Wallbox-Name
			*rscp.NewMessage(rscp.WB_REQ_MODE, nil),               // Uint8 0 = Normal, 0x80 = SunMode
			*rscp.NewMessage(rscp.WB_REQ_STATUS, nil),             // Uint8
			*rscp.NewMessage(rscp.WB_REQ_PM_ACTIVE_PHASES, nil),   // Uint8 7 = 0b111 = 3-Phasen
			*rscp.NewMessage(rscp.WB_REQ_PM_MODE, nil),            // uint8
			*rscp.NewMessage(rscp.WB_REQ_APP_SOFTWARE, nil),       // Uint8
			*rscp.NewMessage(rscp.WB_REQ_KEY_STATE, nil),          // Uint8
			*rscp.NewMessage(rscp.WB_REQ_SUN_MODE_ACTIVE, nil),    // Bool
			*rscp.NewMessage(rscp.WB_REQ_MAX_CHARGE_CURRENT, nil), // Int32 Strom in Amp
			*rscp.NewMessage(rscp.WB_REQ_PROXIMITY_PLUG, nil),     // Int32 Möglicher Strom mit dem Kabel
			*rscp.NewMessage(rscp.WB_REQ_STATION_AVAILABLE, nil),  // Bool
			*rscp.NewMessage(rscp.WB_REQ_CHARGE_FULL, nil),        // Bool
			*rscp.NewMessage(rscp.WB_REQ_PM_ENERGY_L1, nil),       // Float64 L1 Energie in Wh
			*rscp.NewMessage(rscp.WB_REQ_PM_ENERGY_L2, nil),       // Float64 L2 Energie in Wh
			*rscp.NewMessage(rscp.WB_REQ_PM_ENERGY_L3, nil),       // Float64 L3 Energie in Wh
			*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L1, nil),        // Float64 L1 Leistung in W
			*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L2, nil),        // Float64 L2 Leistung in W
			*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L3, nil),        // Float64 L3 Leistung in W
			*rscp.NewMessage(rscp.WB_REQ_DIAG_INFOS, nil),         //
		})
	res, err := m.retryMessages([]rscp.Message{request})
	if err != nil {
		return [3]float64{0}, 0, err
	}

	// Do for FLOATS
	tags, values, err := rscpValuesWithTag(res, cast.ToFloat64E)
	if err != nil {
		return [3]float64{0}, 0, err
	}
	for tag_idx := range tags {
		switch tags[tag_idx] {
		case rscp.WB_INDEX:
		case rscp.WB_ENERGY_ALL:
		case rscp.WB_ENERGY_SOLAR:
		case rscp.WB_SOC:
		case rscp.WB_ERROR_CODE:
		case rscp.WB_MODE:
		case rscp.WB_STATUS:
		case rscp.WB_PM_ACTIVE_PHASES:
		case rscp.WB_PM_MODE:
		case rscp.WB_APP_SOFTWARE:
		case rscp.WB_KEY_STATE:
		case rscp.WB_SUN_MODE_ACTIVE:
		case rscp.WB_MAX_CHARGE_CURRENT:
		case rscp.WB_PROXIMITY_PLUG:
		case rscp.WB_STATION_AVAILABLE:
		case rscp.WB_CHARGE_FULL:
		case rscp.WB_PM_ENERGY_L1:
			total_energy += values[tag_idx]
		case rscp.WB_PM_ENERGY_L2:
			total_energy += values[tag_idx]
		case rscp.WB_PM_ENERGY_L3:
			total_energy += values[tag_idx]
		case rscp.WB_PM_POWER_L1:
			power[0] = values[tag_idx]
		case rscp.WB_PM_POWER_L2:
			power[1] = values[tag_idx]
		case rscp.WB_PM_POWER_L3:
			power[2] = values[tag_idx]
		case rscp.WB_DIAG_INFOS:
		default:
		}
	}

	return power, total_energy / 1000, nil
}

func (wb *E3dc) TestWallboxExternData() (map[string]uint64, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	outObj := map[string]uint64{}

	request := *rscp.NewMessage(
		rscp.WB_REQ_DATA, []rscp.Message{
			// Link
			*rscp.NewMessage(rscp.WB_INDEX, wb.deviceIdx),
			*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_ALL, nil),
			*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_ALG, nil),
			*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_SUN, nil),
			*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_NET, nil),
		})

	res, err := wb.retryMessages([]rscp.Message{request})
	if err != nil {
		return outObj, err
	}

	for _, m := range res {
		for _, arr_lv1 := range m.Value.([]rscp.Message) {
			tag_lv1 := arr_lv1.Tag
			if tag_lv1 != rscp.WB_INDEX {
				data := []byte{}
				for _, arr_lv2 := range arr_lv1.Value.([]rscp.Message) {
					switch arr_lv2.Tag {
					case rscp.WB_EXTERN_DATA_LEN:
					case rscp.WB_EXTERN_DATA:
						data = []byte(arr_lv2.Value.([]uint8))
					}
				}
				switch tag_lv1 {
				case rscp.WB_EXTERN_DATA_ALG:
					outObj["soc"] = cast.ToUint64(data[0])
					outObj["phases"] = cast.ToUint64(data[1])
					status_byte := data[2]
					outObj["maxChargeCurrent"] = cast.ToUint64(data[3]) // Amps
					status_byte2 := data[4]
					outObj["schukoOn"] = cast.ToUint64((data[5] & 1) != 0)
					outObj["schukoAvail"] = cast.ToUint64((data[5] & 16) != 0)
					outObj["StateCode"] = cast.ToUint64(data[6])
					/*
						Keine Anzeige:
							0x00
						Infos:
							0x10 Schuko nicht möglich, int. LM defekt
							0x11 Schuko nicht möglich, Typ 2 > 16A
							0x12 Schuko nicht möglich, Temp > 70*C
							0x13 Schuko nicht möglich, Notstrombetrieb
							0x14 Schuko nicht möglich, Typ 2 vor Schuko
							0x15 Nicht möglich, Schuko oder Typ2 gesteckt
							0x16 Schuko nicht möglich, extern gesperrt
							0x18 Sonnenmodus nicht möglich, kein ext. LM
							0x19 Ext. Abbruch gewünscht
							0x1A Bereichsüberschreitung
							0x1B Passwort falsch
							0x1C Passwort geändert
						Warnungen:
							0x20 Gehäusetemperatur > 50*C, Derating
							0x21 Interner LM nicht vorhanden
							0x22 FRAM CRC falsch, Default Parameter
							0x23 Flash CRC falsch
							0x24 FRAM CRC falsch, Zählerstände
							0x25 Notstrommode aktiv
							0x26 Negative Leistung am Typ 2
							0x27 T-Sensor nicht kalibriert, Imax 16A
						Fehler:
							0x40 FRAM defekt, Default Parameter
							0x41 Flash defekt
							0x42 CAN defekt
							0x43 Gehäusetemperatur > 70*C, alles aus
							0x44 Ladefehler, CP Pegel im Graubereich
							0x45 Ladefehler, Diode defekt
							0x46 Ladefehler, PP unbekannt
							0x47 Ladegeschirr defekt, PP control
					*/
					outObj["alg_b7"] = cast.ToUint64(data[7])

					outObj["sunModeOn"] = cast.ToUint64((status_byte & 128) != 0)       // Bit 7
					outObj["chargingCanceled"] = cast.ToUint64((status_byte & 64) != 0) // Bit 6
					outObj["chargingActive"] = cast.ToUint64((status_byte & 32) != 0)   // Bit 5
					outObj["plugLocked"] = cast.ToUint64((status_byte & 16) != 0)       // Bit 4
					outObj["plugged"] = cast.ToUint64((status_byte & 8) != 0)           // Bit 3
					outObj["schukoOn"] = cast.ToUint64((status_byte & 4) != 0)          // Bit 2
					outObj["schukoPlugged"] = cast.ToUint64((status_byte & 2) != 0)     // Bit 1
					outObj["schukoLocked"] = cast.ToUint64((status_byte & 1) != 0)      // Bit 0

					outObj["LedErr"] = cast.ToUint64((status_byte2 & 128) != 0)              // Bit 7
					outObj["LedSun"] = cast.ToUint64((status_byte2 & 64) != 0)               // Bit 6
					outObj["LedGreen"] = cast.ToUint64((status_byte2 & 32) != 0)             // Bit 5
					outObj["relais16amps1pSchuko"] = cast.ToUint64((status_byte2 & 16) != 0) // Bit 4
					outObj["relais16amps3p"] = cast.ToUint64((status_byte2 & 8) != 0)        // Bit 3
					outObj["relais32amp3p"] = cast.ToUint64((status_byte2 & 4) != 0)         // Bit 2
					outObj["reqCarFanOn"] = cast.ToUint64((status_byte2 & 2) != 0)           // Bit 1
				case rscp.WB_EXTERN_DATA_SUN:
					outObj["sun_power"] = uint64(data[1])<<8 + uint64(data[0])
					outObj["sun_energy"] = uint64(data[5])<<24 + uint64(data[4])<<16 + uint64(data[3])<<8 + uint64(data[2]) // kWh
					outObj["sun_fraction"] = cast.ToUint64(data[6])
					outObj["sun_b7"] = cast.ToUint64(data[7])
				case rscp.WB_EXTERN_DATA_NET:
					outObj["net_power"] = uint64(data[1])<<8 + uint64(data[0])
					outObj["net_energy"] = uint64(data[5])<<24 + uint64(data[4])<<16 + uint64(data[3])<<8 + uint64(data[2]) // kWh
					outObj["net_fraction"] = cast.ToUint64(data[6])
					outObj["net_b7"] = cast.ToUint64(data[7])
				case rscp.WB_EXTERN_DATA_ALL:
					outObj["all_power"] = uint64(data[1])<<8 + uint64(data[0])
					outObj["all_energy"] = uint64(data[5])<<24 + uint64(data[4])<<16 + uint64(data[3])<<8 + uint64(data[2]) // kWh
					outObj["all_fraction"] = cast.ToUint64(data[6])
					outObj["all_b7"] = cast.ToUint64(data[7])
				default:
				}
			}
		}
	}
	return outObj, nil
}
func (wb *E3dc) TestWallboxSetExternData(current, mode, switchPases, cancelCharging uint8) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	var newData = make([]uint8, 6)
	newData[0] = current        // current in amps
	newData[1] = mode           // 1 = SunMode, 2 = Mixed
	newData[2] = 0              //
	newData[3] = switchPases    // 1 = Phasen-Wechseln
	newData[4] = cancelCharging // 1 = Cancel Charging
	newData[5] = 0              //

	request := *rscp.NewMessage(
		rscp.WB_REQ_DATA, []rscp.Message{
			*rscp.NewMessage(rscp.WB_INDEX, wb.deviceIdx),
			*rscp.NewMessage(rscp.WB_REQ_SET_EXTERN, []rscp.Message{
				*rscp.NewMessage(rscp.WB_EXTERN_DATA_LEN, uint8(6)),
				*rscp.NewMessage(rscp.WB_EXTERN_DATA, newData)}),
		})

	res, err := wb.retryMessages([]rscp.Message{request})
	_, _, err = rscpValuesWithTag(res, cast.ToInt32E)

	return err
}

func (wb *E3dc) TestWallboxTags() error {
	// Just availability of WB, no charging info
	res2, err2 := ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_DEVICE_STATE, nil)}, []rscp.Tag{rscp.WB_DEVICE_CONNECTED, rscp.WB_DEVICE_WORKING, rscp.WB_DEVICE_IN_SERVICE}, cast.ToInt64E)
	if err2 != nil {
		//return false, err 1,1,0
	}
	res2 = append(res2, -1) // 1, 1, 0

	res3, err3 := ReadComplexTags(wb, []rscp.Message{*rscp.NewMessage(rscp.WB_REQ_STATUS, nil)}, []rscp.Tag{rscp.WB_STATUS}, cast.ToUint8E) // No info if plugged in!
	if err3 != nil {
		return err3
	}
	res3 = append(res3, 1) // 0

	return nil
}
