package charger

import (
	"errors"
	"fmt"
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
	conn    *rscp.Client
	id      uint8
	current byte
}

func init() {
	registry.Add("e3dc-rscp", NewE3dcFromConfig)
}

func NewE3dcFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Uri      string
		User     string
		Password string
		Key      string
		Id       uint8
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

	return NewE3dc(cfg, cc.Id)
}

var e3dcOnce sync.Once

func NewE3dc(cfg rscp.ClientConfig, id uint8) (api.Charger, error) {
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
		conn:    conn,
		id:      id,
		current: 6,
	}

	return m, nil
}

// Enabled implements the api.Charger interface
func (wb *E3dc) Enabled() (bool, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_ALG, nil),
	}))
	if err != nil {
		return false, err
	}
	fmt.Printf("%+v\n", res)

	wb_data, err := rscpContainer(*res, 2)
	if err != nil {
		return false, err
	}

	wb_ext_data_alg, err := rscpContainer(wb_data[1], 2)
	if err != nil {
		return false, err
	}

	b, err := rscpBytes(wb_ext_data_alg[1])
	if err != nil {
		return false, err
	}

	// WB_EXTERN_DATA_ALG byte structure (tested 2025-11-28):
	// Byte 2, Bit 6: 0 = enabled, 1 = disabled (abort active)
	// Example: b[2] = 4 (0b00000100) -> enabled
	//          b[2] = 68 (0b01000100) -> disabled (bit 6 set)
	enabled := b[2]&(1<<6) == 0
	fmt.Printf("Enabled() -> b[2]=%d, enabled=%v\n", b[2], enabled)
	return enabled, nil
}

// Enable implements the api.Charger interface
func (wb *E3dc) Enable(enable bool) error {
	// WB_REQ_SET_ABORT_CHARGING controls charging:
	//   true  = abort/stop charging
	//   false = allow/resume charging
	abort := !enable
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SET_ABORT_CHARGING, abort),
	}))

	fmt.Printf("Enable() -> enable=%v, abort=%v, res=%+v, err=%v\n", enable, abort, res, err)
	return err
}

// Status implements the api.Charger interface
func (wb *E3dc) Status() (api.ChargeStatus, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_ALG, nil),
	}))
	if err != nil {
		return api.StatusNone, err
	}

	wb_data, err := rscpContainer(*res, 2)
	if err != nil {
		return api.StatusNone, err
	}

	wb_ext_data_alg, err := rscpContainer(wb_data[1], 2)
	if err != nil {
		return api.StatusNone, err
	}

	b, err := rscpBytes(wb_ext_data_alg[1])
	if err != nil {
		return api.StatusNone, err
	}

	// WB_EXTERN_DATA_ALG byte 2 contains charging state bits:
	//   Bit 2 (0x04): wallbox available, no vehicle connected
	//   Bit 3 (0x08): vehicle connected
	//   Bit 5 (0x20): charging active
	//   Bit 6 (0x40): charging paused/inhibited
	//
	// Tested values (2025-11-28):
	//   StatusA (no car):      b[2] =  4 (0b00000100) - bit 2 set
	//   StatusB (paused):      b[2] = 72 (0b01001000) - bits 3,6 set
	//   StatusC (charging):    b[2] = 40 (0b00101000) - bits 3,5 set
	switch {
	case b[2]&0x20 != 0: // bit 5: charging active
		return api.StatusC, nil
	case b[2]&0x08 != 0: // bit 3: vehicle connected
		return api.StatusB, nil
	default:
		return api.StatusA, nil
	}
}

func (wb *E3dc) maxCurrent(current int64) error {
	// WB_REQ_SET_MAX_CHARGE_CURRENT sets the charging current limit in Ampere
	// Supported range: 6-32A in 1A steps (UChar8)
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SET_MAX_CHARGE_CURRENT, uint8(current)),
	}))

	fmt.Printf("maxCurrent() -> current=%dA, res=%+v, err=%v\n", current, res, err)

	if err == nil {
		wb.current = byte(current)
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *E3dc) MaxCurrent(current int64) error {
	return wb.maxCurrent(current)
}

var _ api.Meter = (*E3dc)(nil)

// CurrentPower implements the api.Meter interface
func (wb *E3dc) CurrentPower() (float64, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L1, nil),
		*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L2, nil),
		*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L3, nil),
	}))
	if err != nil {
		return 0, err
	}

	wb_data, err := rscpContainer(*res, 4)
	if err != nil {
		return 0, err
	}

	var power float64
	for i := 1; i <= 3; i++ {
		p, err := rscpFloat64(wb_data[i])
		if err != nil {
			return 0, err
		}
		power += p
	}

	fmt.Printf("CurrentPower() -> %.1f W\n", power)
	return power, nil
}

var _ api.MeterEnergy = (*E3dc)(nil)

// TotalEnergy implements the api.MeterEnergy interface
//
// E3DC stores wallbox energy in two separate counters that must be added:
//   - DB_TEC_WALLBOX_ENERGYALL: Historical energy stored in the database (persisted)
//   - WB_ENERGY_ALL: Energy since last database sync (volatile, resets on sync)
//
// The sum of both values matches the total energy shown in the E3DC portal.
// Testing showed: DB_TEC (8319 kWh) + WB_ENERGY (699 kWh) = 9018 kWh ≈ Portal (9019 kWh)
func (wb *E3dc) TotalEnergy() (float64, error) {
	// Query both energy sources in parallel
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.DB_REQ_TEC_WALLBOX_VALUES, nil))
	if err != nil {
		return 0, err
	}

	// Parse DB_TEC_WALLBOX_VALUES response
	// Structure: DB_TEC_WALLBOX_VALUES -> DB_TEC_WALLBOX_VALUES -> []DB_TEC_WALLBOX_VALUE
	// Each DB_TEC_WALLBOX_VALUE contains: DB_TEC_WALLBOX_INDEX, DB_TEC_WALLBOX_ENERGYALL, DB_TEC_WALLBOX_WB_ENERGY_SOLAR
	outer, err := rscpContainer(*res, 1)
	if err != nil {
		return 0, err
	}

	inner, err := rscpContainer(outer[0], 1)
	if err != nil {
		return 0, err
	}

	// Find the wallbox with matching index
	var dbEnergy float64
	for _, wbValue := range inner {
		wbData, err := rscpContainer(wbValue, 3)
		if err != nil {
			continue
		}

		idx, err := rscpUint8(wbData[0])
		if err != nil || idx != wb.id {
			continue
		}

		dbEnergy, err = rscpFloat64(wbData[1])
		if err != nil {
			return 0, err
		}
		break
	}

	// Query WB_ENERGY_ALL for energy since last DB sync
	res, err = wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_ENERGY_ALL, nil),
	}))
	if err != nil {
		return 0, err
	}

	wb_data, err := rscpContainer(*res, 2)
	if err != nil {
		return 0, err
	}

	wbEnergy, err := rscpFloat64(wb_data[1])
	if err != nil {
		return 0, err
	}

	// Sum both counters and convert Wh to kWh
	totalWh := dbEnergy + wbEnergy
	kWh := totalWh / 1000.0
	fmt.Printf("TotalEnergy() -> DB_TEC=%.0f Wh + WB=%.0f Wh = %.0f Wh -> %.3f kWh\n", dbEnergy, wbEnergy, totalWh, kWh)
	return kWh, nil
}

var _ api.PhasePowers = (*E3dc)(nil)

// Powers implements the api.PhasePowers interface
func (wb *E3dc) Powers() (float64, float64, float64, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L1, nil),
		*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L2, nil),
		*rscp.NewMessage(rscp.WB_REQ_PM_POWER_L3, nil),
	}))
	if err != nil {
		return 0, 0, 0, err
	}

	wb_data, err := rscpContainer(*res, 4)
	if err != nil {
		return 0, 0, 0, err
	}

	p1, err := rscpFloat64(wb_data[1])
	if err != nil {
		return 0, 0, 0, err
	}

	p2, err := rscpFloat64(wb_data[2])
	if err != nil {
		return 0, 0, 0, err
	}

	p3, err := rscpFloat64(wb_data[3])
	if err != nil {
		return 0, 0, 0, err
	}

	fmt.Printf("Powers() -> L1=%.1f W, L2=%.1f W, L3=%.1f W\n", p1, p2, p3)
	return p1, p2, p3, nil
}

var _ api.PhaseCurrents = (*E3dc)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *E3dc) Currents() (float64, float64, float64, error) {
	// E3DC doesn't provide current directly, calculate from power
	// Assume 230V nominal voltage
	p1, p2, p3, err := wb.Powers()
	if err != nil {
		return 0, 0, 0, err
	}

	const voltage = 230.0
	i1 := p1 / voltage
	i2 := p2 / voltage
	i3 := p3 / voltage

	fmt.Printf("Currents() -> L1=%.2f A, L2=%.2f A, L3=%.2f A\n", i1, i2, i3)
	return i1, i2, i3, nil
}

var _ api.PhaseGetter = (*E3dc)(nil)

// GetPhases implements the api.PhaseGetter interface
func (wb *E3dc) GetPhases() (int, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_PM_ACTIVE_PHASES, nil),
	}))
	if err != nil {
		return 0, err
	}

	wb_data, err := rscpContainer(*res, 2)
	if err != nil {
		return 0, err
	}

	// WB_PM_ACTIVE_PHASES returns bitmask: bit0=L1, bit1=L2, bit2=L3
	phaseMask, err := rscpUint8(wb_data[1])
	if err != nil {
		return 0, err
	}

	// Count active phases from bitmask
	phases := 0
	if phaseMask&0x01 != 0 {
		phases++
	}
	if phaseMask&0x02 != 0 {
		phases++
	}
	if phaseMask&0x04 != 0 {
		phases++
	}

	fmt.Printf("GetPhases() -> mask=0x%02x, phases=%d\n", phaseMask, phases)
	return phases, nil
}

var _ api.CurrentLimiter = (*E3dc)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (wb *E3dc) GetMinMaxCurrent() (float64, float64, error) {
	// E3DC wallbox supports 6-32A per phase (Type 2)
	return 6, 32, nil
}

var _ api.CurrentGetter = (*E3dc)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *E3dc) GetMaxCurrent() (float64, error) {
	// Return cached current value
	return float64(wb.current), nil
}

// var _ api.Identifier = (*E3dc)(nil)

// // Identify implements the api.Identifier interface
// func (wb *E3dc) Identify() (string, error) {
// 	return "", api.ErrNotAvailable
// }

// var _ api.PhaseSwitcher = (*E3dc)(nil)

// // Phases1p3p implements the api.PhaseSwitcher interface
// func (wb *E3dc) Phases1p3p(phases int) error {
// 	return api.ErrNotAvailable
// }

func rscpError(msg ...rscp.Message) error {
	var errs []error
	for _, m := range msg {
		if m.DataType == rscp.Error {
			errs = append(errs, errors.New(rscp.RscpError(cast.ToUint32(m.Value)).String()))
		}
	}
	return errors.Join(errs...)
}

func rscpContainer(msg rscp.Message, length int) ([]rscp.Message, error) {
	if err := rscpError(msg); err != nil {
		return nil, err
	}

	if msg.DataType != rscp.Container {
		return nil, errors.New("invalid response")
	}

	res, ok := msg.Value.([]rscp.Message)
	if !ok {
		return nil, errors.New("invalid response")
	}

	if l := len(res); l < length {
		return nil, fmt.Errorf("invalid length: expected %d, got %d", length, l)
	}

	return res, nil
}

func rscpBytes(msg rscp.Message) ([]byte, error) {
	return rscpValue(msg, func(data any) ([]byte, error) {
		b, ok := data.([]uint8)
		if !ok {
			return nil, errors.New("invalid response")
		}
		return b, nil
	})
}

func rscpFloat64(msg rscp.Message) (float64, error) {
	return rscpValue(msg, func(data any) (float64, error) {
		return cast.ToFloat64E(data)
	})
}

func rscpUint8(msg rscp.Message) (uint8, error) {
	return rscpValue(msg, func(data any) (uint8, error) {
		return cast.ToUint8E(data)
	})
}

func rscpValue[T any](msg rscp.Message, fun func(any) (T, error)) (T, error) {
	var zero T
	if err := rscpError(msg); err != nil {
		return zero, err
	}

	return fun(msg.Value)
}

// func rscpValues[T any](msg []rscp.Message, fun func(any) (T, error)) ([]T, error) {
// 	res := make([]T, 0, len(msg))

// 	for _, m := range msg {
// 		v, err := rscpValue(m, fun)
// 		if err != nil {
// 			return nil, err
// 		}

// 		res = append(res, v)
// 	}

// 	return res, nil
// }
