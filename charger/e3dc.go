package charger

// E3DC Multi Connect II Wallbox Charger
//
// REQUIREMENTS - Configure the following in the E3DC portal for evcc control:
//   - Sun Mode (Sonnenmodus): OFF - will be disabled automatically at startup
//   - Auto Phase Switching: OFF - required if evcc should control 1p/3p switching
//   - Charge Authorization: OFF or configure RFID - evcc needs to control charging
//
// Sun mode is automatically disabled at startup. Auto phase switching generates
// a warning but is not changed automatically (user may want to keep it).
//
// DEVELOPMENT STATUS:
// - Tested with E3DC Multi Connect II Wallbox (FW 7.0.6.0/1.0.3.0)
// - Phase switching (1p3p): E3DC handles ramping internally (tested)
// - Requires testing with additional E3DC systems before production use

import (
	"context"
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
	registry.AddCtx("e3dc-rscp", NewE3dcFromConfig)
}

// NewE3dcFromConfig creates an E3DC charger from generic config
func NewE3dcFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
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

	host, portStr, err := net.SplitHostPort(util.DefaultPort(cc.Uri, 5033))
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

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

	return NewE3dc(ctx, cfg, cc.Id)
}

var e3dcOnce sync.Once

// NewE3dc creates E3DC charger
func NewE3dc(ctx context.Context, cfg rscp.ClientConfig, id uint8) (*E3dc, error) {
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

	// Check wallbox configuration and warn if not optimal for evcc control
	wb.checkConfiguration()

	return wb, nil
}

// checkConfiguration verifies wallbox settings and adjusts them for evcc control
func (wb *E3dc) checkConfiguration() {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SUN_MODE_ACTIVE, nil),
		*rscp.NewMessage(rscp.WB_REQ_AUTO_PHASE_SWITCH_ENABLED, nil),
	}))
	if err != nil {
		wb.log.WARN.Printf("failed to query wallbox configuration: %v", err)
		return
	}

	wbData, err := rscpContainer(*res, 3)
	if err != nil {
		wb.log.WARN.Printf("failed to parse wallbox configuration: %v", err)
		return
	}

	// Check and disable sun mode - evcc needs to control charging
	if sunMode, err := rscpValue(wbData[1], func(data any) (bool, error) {
		if val, ok := data.(bool); ok {
			return val, nil
		}
		return false, errors.New("invalid type")
	}); err == nil && sunMode {
		wb.log.WARN.Println("wallbox sun mode is enabled - disabling for evcc control")
		if _, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
			*rscp.NewMessage(rscp.WB_INDEX, wb.id),
			*rscp.NewMessage(rscp.WB_REQ_SET_SUN_MODE_ACTIVE, false),
		})); err != nil {
			wb.log.ERROR.Printf("failed to disable sun mode: %v", err)
		}
	}

	// Warn about auto phase switching - user may want to keep it for non-evcc use
	if autoPhase, err := rscpValue(wbData[2], func(data any) (bool, error) {
		if val, ok := data.(bool); ok {
			return val, nil
		}
		return false, errors.New("invalid type")
	}); err == nil && autoPhase {
		wb.log.WARN.Println("wallbox auto phase switching is enabled - disable in E3DC portal if you want evcc to control 1p/3p switching")
	}
}

// getExternDataAlg retrieves the WB_EXTERN_DATA_ALG status byte array
func (wb *E3dc) getExternDataAlg() ([]byte, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_ALG, nil),
	}))
	if err != nil {
		return nil, err
	}

	wbData, err := rscpContainer(*res, 2)
	if err != nil {
		return nil, err
	}

	wbExtDataAlg, err := rscpContainer(wbData[1], 2)
	if err != nil {
		return nil, err
	}

	b, err := rscpBytes(wbExtDataAlg[1])
	if err != nil {
		return nil, err
	}

	if len(b) < 3 {
		return nil, fmt.Errorf("invalid WB_EXTERN_DATA_ALG length: %d", len(b))
	}

	return b, nil
}

// Enabled implements the api.Charger interface
func (wb *E3dc) Enabled() (bool, error) {
	b, err := wb.getExternDataAlg()
	if err != nil {
		return false, err
	}

	// WB_EXTERN_DATA_ALG Byte 2, Bit 6 (0b01000000): 0 = enabled, 1 = disabled (abort active)
	return b[2]&0b01000000 == 0, nil
}

// Enable implements the api.Charger interface
// Controls charging by setting the abort flag (inverted logic: abort=false means enabled)
func (wb *E3dc) Enable(enable bool) error {
	_, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SET_ABORT_CHARGING, !enable),
	}))

	return err
}

// Status implements the api.Charger interface
// Returns the charging state by reading status flags from WB_EXTERN_DATA_ALG
func (wb *E3dc) Status() (api.ChargeStatus, error) {
	b, err := wb.getExternDataAlg()
	if err != nil {
		return api.StatusNone, err
	}

	// WB_EXTERN_DATA_ALG Byte 2 status bits:
	//   Bit 5 (0b00100000): Charging active
	//   Bit 3 (0b00001000): Vehicle connected
	//   Bit 2 (0b00000100): Ready, no vehicle
	//
	// IMPORTANT: Check order is C→B→A (not A→B→C) because bits are not mutually exclusive!
	// When charging (StatusC), both Bit 5 AND Bit 3 are set (e.g., 0b00101000).
	// We must check Bit 5 first, otherwise 0b00101000 would incorrectly match StatusB.
	//
	// Explicitly checking Bit 2 for StatusA allows us to detect error states
	// like 0b00000000 (no bits set) or 0b01000000 (only disabled flag).
	switch {
	case b[2]&0b00100000 != 0: // Bit 5: charging active → StatusC
		return api.StatusC, nil
	case b[2]&0b00001000 != 0: // Bit 3: vehicle connected → StatusB
		return api.StatusB, nil
	case b[2]&0b00000100 != 0: // Bit 2: ready, no vehicle → StatusA
		return api.StatusA, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown wallbox status: 0x%02x", b[2])
	}
}

// MaxCurrent implements the api.Charger interface
func (wb *E3dc) MaxCurrent(current int64) error {
	_, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SET_MAX_CHARGE_CURRENT, uint8(current)),
	}))

	return err
}

var _ api.Meter = (*E3dc)(nil)

// CurrentPower implements the api.Meter interface
// Returns the total charging power by summing all three phases
func (wb *E3dc) CurrentPower() (float64, error) {
	p1, p2, p3, err := wb.powers()
	if err != nil {
		return 0, err
	}

	return p1 + p2 + p3, nil
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
	// Query both energy sources sequentially
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

// powers returns the charging power for each individual phase in watts
func (wb *E3dc) powers() (float64, float64, float64, error) {
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
// Calculates current from power readings as voltage readings are not accessible
func (wb *E3dc) Currents() (float64, float64, float64, error) {
	p1, p2, p3, err := wb.powers()
	if err != nil {
		return 0, 0, 0, err
	}

	// Calculate current from power using nominal 230V
	// Note: WB_REQ_DIAG_PHASE_VOLTAGE returns ERR_ACCESS_DENIED
	const voltage = 230.0
	i1 := p1 / voltage
	i2 := p2 / voltage
	i3 := p3 / voltage

	return i1, i2, i3, nil
}

var _ api.PhaseGetter = (*E3dc)(nil)

// GetPhases implements the api.PhaseGetter interface
// Returns the configured number of phases (1 or 3)
// Note: WB_PM_ACTIVE_PHASES reports physical wiring, WB_NUMBER_PHASES reports actual configuration
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
// Returns the wallbox's hardware current limits (typically 6-32A)
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
// Returns the currently configured maximum charging current
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

// getSessionData retrieves the session data container
func (wb *E3dc) getSessionData() ([]rscp.Message, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_SESSION, nil))
	if err != nil {
		return nil, err
	}

	return rscpContainer(*res, 1)
}

var _ api.ChargeRater = (*E3dc)(nil)

// ChargedEnergy implements the api.ChargeRater interface
// Returns the energy charged in the current session from WB_SESSION_CHARGED_ENERGY
func (wb *E3dc) ChargedEnergy() (float64, error) {
	sessionData, err := wb.getSessionData()
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
// Returns the active charging duration from WB_SESSION_ACTIVE_CHARGE_TIME
func (wb *E3dc) ChargeDuration() (time.Duration, error) {
	sessionData, err := wb.getSessionData()
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
// Returns the RFID tag ID from WB_SESSION_AUTH_DATA if a session is active
func (wb *E3dc) Identify() (string, error) {
	sessionData, err := wb.getSessionData()
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
// Switches between 1-phase and 3-phase charging
// The wallbox handles the safe switching sequence internally (reduce current, switch, ramp up)
// Requirements: WB_AUTO_PHASE_SWITCH_ENABLED must be disabled in the E3DC dashboard
func (wb *E3dc) Phases1p3p(phases int) error {
	if phases != 1 && phases != 3 {
		return fmt.Errorf("invalid phases: %d (must be 1 or 3)", phases)
	}

	// Check if automatic phase switching is disabled (required for manual control)
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_AUTO_PHASE_SWITCH_ENABLED, nil),
	}))
	if err != nil {
		return err
	}

	wbData, err := rscpContainer(*res, 2)
	if err != nil {
		return err
	}

	autoPhaseSwitch, err := rscpValue(wbData[1], func(data any) (bool, error) {
		val, ok := data.(bool)
		if !ok {
			return false, errors.New("invalid auto phase switch response")
		}
		return val, nil
	})
	if err != nil {
		return err
	}

	if autoPhaseSwitch {
		return errors.New("automatic phase switching is enabled - please disable it in the E3DC dashboard to allow manual phase control")
	}

	// Perform phase switch
	_, err = wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SET_NUMBER_PHASES, uint8(phases)),
	}))

	return err
}

// rscpError extracts error messages from RSCP responses
func rscpError(msg ...rscp.Message) error {
	var errs []error
	for _, m := range msg {
		if m.DataType == rscp.Error {
			errs = append(errs, errors.New(rscp.RscpError(cast.ToUint32(m.Value)).String()))
		}
	}
	return errors.Join(errs...)
}

// rscpContainer extracts and validates a container message
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
		return nil, fmt.Errorf("invalid length: expected at least %d, got %d", length, l)
	}

	return res, nil
}

// rscpBytes extracts a byte array from an RSCP message
func rscpBytes(msg rscp.Message) ([]byte, error) {
	return rscpValue(msg, func(data any) ([]byte, error) {
		b, ok := data.([]uint8)
		if !ok {
			return nil, errors.New("invalid response")
		}
		return b, nil
	})
}

// rscpFloat64 extracts a float64 value from an RSCP message
func rscpFloat64(msg rscp.Message) (float64, error) {
	return rscpValue(msg, func(data any) (float64, error) {
		return cast.ToFloat64E(data)
	})
}

// rscpUint8 extracts a uint8 value from an RSCP message
func rscpUint8(msg rscp.Message) (uint8, error) {
	return rscpValue(msg, func(data any) (uint8, error) {
		return cast.ToUint8E(data)
	})
}

// rscpString extracts a string value from an RSCP message
func rscpString(msg rscp.Message) (string, error) {
	return rscpValue(msg, func(data any) (string, error) {
		return cast.ToStringE(data)
	})
}

// rscpValue is a generic helper for extracting typed values from RSCP messages
func rscpValue[T any](msg rscp.Message, fun func(any) (T, error)) (T, error) {
	var zero T
	if err := rscpError(msg); err != nil {
		return zero, err
	}

	return fun(msg.Value)
}
