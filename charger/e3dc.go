package charger

// DEVELOPMENT STATUS:
// - Tested with E3DC Multi Connect II Wallbox (FW 7.0.6.0/1.0.3.0)
// - Individual RSCP calls verified, full evcc integration pending
// - Phase switching (1p3p): E3DC handles ramping internally (tested)
// - Requires testing with additional E3DC systems before production use

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

// E3dc charger implementation using RSCP protocol
type E3dc struct {
	log  *util.Logger
	conn *rscp.Client
	id   uint8
}

func init() {
	registry.Add("e3dc-rscp", NewE3dcFromConfig)
}

// NewE3dcFromConfig creates an E3DC charger from generic config
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

// NewE3dc creates E3DC charger
func NewE3dc(cfg rscp.ClientConfig, id uint8) (*E3dc, error) {
	log := util.NewLogger("e3dc")

	e3dcOnce.Do(func() {
		rscp.Log.SetLevel(logrus.DebugLevel)
		rscp.Log.SetOutput(log.TRACE.Writer())
	})

	conn, err := rscp.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	wb := &E3dc{
		log:  log,
		conn: conn,
		id:   id,
	}

	return wb, nil
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

	wbData, err := rscpContainer(*res, 2)
	if err != nil {
		return false, err
	}

	wbExtDataAlg, err := rscpContainer(wbData[1], 2)
	if err != nil {
		return false, err
	}

	b, err := rscpBytes(wbExtDataAlg[1])
	if err != nil {
		return false, err
	}

	// WB_EXTERN_DATA_ALG Byte 2, Bit 6: 0 = enabled, 1 = disabled (abort active)
	return b[2]&(1<<6) == 0, nil
}

// Enable implements the api.Charger interface
func (wb *E3dc) Enable(enable bool) error {
	// WB_REQ_SET_ABORT_CHARGING: true = abort/stop, false = allow/resume
	_, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SET_ABORT_CHARGING, !enable),
	}))

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

	wbData, err := rscpContainer(*res, 2)
	if err != nil {
		return api.StatusNone, err
	}

	wbExtDataAlg, err := rscpContainer(wbData[1], 2)
	if err != nil {
		return api.StatusNone, err
	}

	b, err := rscpBytes(wbExtDataAlg[1])
	if err != nil {
		return api.StatusNone, err
	}

	// WB_EXTERN_DATA_ALG Byte 2: Bit 5 (0x20) = charging, Bit 3 (0x08) = connected
	switch {
	case b[2]&0x20 != 0:
		return api.StatusC, nil
	case b[2]&0x08 != 0:
		return api.StatusB, nil
	default:
		return api.StatusA, nil
	}
}

func (wb *E3dc) maxCurrent(current int64) error {
	_, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SET_MAX_CHARGE_CURRENT, uint8(current)),
	}))

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

	wbData, err := rscpContainer(*res, 4)
	if err != nil {
		return 0, err
	}

	var power float64
	for i := 1; i <= 3; i++ {
		p, err := rscpFloat64(wbData[i])
		if err != nil {
			return 0, err
		}
		power += p
	}

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

	wbData, err := rscpContainer(*res, 2)
	if err != nil {
		return 0, err
	}

	wbEnergy, err := rscpFloat64(wbData[1])
	if err != nil {
		return 0, err
	}

	// Sum both counters and convert Wh to kWh
	return (dbEnergy + wbEnergy) / 1000.0, nil
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

	wbData, err := rscpContainer(*res, 4)
	if err != nil {
		return 0, 0, 0, err
	}

	p1, err := rscpFloat64(wbData[1])
	if err != nil {
		return 0, 0, 0, err
	}

	p2, err := rscpFloat64(wbData[2])
	if err != nil {
		return 0, 0, 0, err
	}

	p3, err := rscpFloat64(wbData[3])
	if err != nil {
		return 0, 0, 0, err
	}

	return p1, p2, p3, nil
}

var _ api.PhaseCurrents = (*E3dc)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *E3dc) Currents() (float64, float64, float64, error) {
	p1, p2, p3, err := wb.Powers()
	if err != nil {
		return 0, 0, 0, err
	}

	// Calculate current from power assuming 230V nominal voltage
	// E3DC WB_REQ_DIAG_PHASE_VOLTAGE returns ERR_ACCESS_DENIED
	const voltage = 230.0
	i1 := p1 / voltage
	i2 := p2 / voltage
	i3 := p3 / voltage

	return i1, i2, i3, nil
}

var _ api.PhaseGetter = (*E3dc)(nil)

// GetPhases implements the api.PhaseGetter interface
// Returns the configured number of phases (1 or 3)
// Note: WB_PM_ACTIVE_PHASES always returns 7 (physical wiring), so we use WB_NUMBER_PHASES instead
func (wb *E3dc) GetPhases() (int, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_NUMBER_PHASES, nil),
	}))
	if err != nil {
		return 0, err
	}

	wbData, err := rscpContainer(*res, 2)
	if err != nil {
		return 0, err
	}

	phases, err := rscpUint8(wbData[1])
	if err != nil {
		return 0, err
	}

	return int(phases), nil
}

var _ api.CurrentLimiter = (*E3dc)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (wb *E3dc) GetMinMaxCurrent() (float64, float64, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_LOWER_CURRENT_LIMIT, nil),
		*rscp.NewMessage(rscp.WB_REQ_UPPER_CURRENT_LIMIT, nil),
	}))
	if err != nil {
		return 0, 0, err
	}

	wbData, err := rscpContainer(*res, 3)
	if err != nil {
		return 0, 0, err
	}

	minCurrent, err := rscpFloat64(wbData[1])
	if err != nil {
		return 0, 0, err
	}

	maxCurrent, err := rscpFloat64(wbData[2])
	if err != nil {
		return 0, 0, err
	}

	return minCurrent, maxCurrent, nil
}

var _ api.CurrentGetter = (*E3dc)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *E3dc) GetMaxCurrent() (float64, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_MAX_CHARGE_CURRENT, nil),
	}))
	if err != nil {
		return 0, err
	}

	wbData, err := rscpContainer(*res, 2)
	if err != nil {
		return 0, err
	}

	current, err := rscpFloat64(wbData[1])
	if err != nil {
		return 0, err
	}

	return current, nil
}

var _ api.ChargeRater = (*E3dc)(nil)

// ChargedEnergy implements the api.ChargeRater interface
// Returns the energy charged in the current session in kWh
func (wb *E3dc) ChargedEnergy() (float64, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_SESSION, nil))
	if err != nil {
		return 0, err
	}

	sessionData, err := rscpContainer(*res, 1)
	if err != nil {
		return 0, err
	}

	// Find WB_SESSION_CHARGED_ENERGY in session data
	for _, msg := range sessionData {
		if msg.Tag == rscp.WB_SESSION_CHARGED_ENERGY {
			energy, err := rscpFloat64(msg)
			if err != nil {
				return 0, err
			}
			return energy / 1000.0, nil // Wh -> kWh
		}
	}

	// No active session
	return 0, nil
}

var _ api.ChargeTimer = (*E3dc)(nil)

// ChargeDuration implements the api.ChargeTimer interface
// Returns the duration of the current charging session
func (wb *E3dc) ChargeDuration() (time.Duration, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_SESSION, nil))
	if err != nil {
		return 0, err
	}

	sessionData, err := rscpContainer(*res, 1)
	if err != nil {
		return 0, err
	}

	// Find WB_SESSION_ACTIVE_CHARGE_TIME in session data
	for _, msg := range sessionData {
		if msg.Tag == rscp.WB_SESSION_ACTIVE_CHARGE_TIME {
			// Session time is in milliseconds (Uint64)
			ms, err := rscpFloat64(msg)
			if err != nil {
				return 0, err
			}
			return time.Duration(ms) * time.Millisecond, nil
		}
	}

	// No active session
	return 0, nil
}

var _ api.Identifier = (*E3dc)(nil)

// Identify implements the api.Identifier interface
// Returns the RFID tag ID used to authorize the current session
func (wb *E3dc) Identify() (string, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_SESSION, nil))
	if err != nil {
		return "", err
	}

	sessionData, err := rscpContainer(*res, 1)
	if err != nil {
		return "", err
	}

	// Find WB_SESSION_AUTH_DATA in session data
	for _, msg := range sessionData {
		if msg.Tag == rscp.WB_SESSION_AUTH_DATA {
			return rscpString(msg)
		}
	}

	// No active session or no RFID used
	return "", nil
}

var _ api.PhaseSwitcher = (*E3dc)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
// The E3DC wallbox handles the safe switching internally:
// 1. Reduces current to 0
// 2. Switches phases
// 3. Ramps current back up
// Requires WB_AUTO_PHASE_SWITCH_ENABLED to be disabled in the E3DC dashboard
func (wb *E3dc) Phases1p3p(phases int) error {
	if phases != 1 && phases != 3 {
		return fmt.Errorf("invalid phases: %d (must be 1 or 3)", phases)
	}

	_, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SET_NUMBER_PHASES, uint8(phases)),
	}))

	return err
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

func rscpString(msg rscp.Message) (string, error) {
	return rscpValue(msg, func(data any) (string, error) {
		return cast.ToStringE(data)
	})
}

func rscpValue[T any](msg rscp.Message, fun func(any) (T, error)) (T, error) {
	var zero T
	if err := rscpError(msg); err != nil {
		return zero, err
	}

	return fun(msg.Value)
}
