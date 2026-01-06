package charger

// E3DC Wallbox Charger (RSCP Protocol)
//
// REQUIREMENTS - Configure in E3DC portal for evcc control:
//   - Sun Mode (Sonnenmodus): OFF
//   - Auto Phase Switching: OFF
//   - Charge Authorization: OFF or configure RFID
//
// evcc will automatically disable Sun Mode and Auto Phase Switching at startup
// if still enabled, but the user should configure this in the E3DC portal.
//
// TESTED WITH:
//   - E3DC Multi Connect II Wallbox (FW 7.0.6.0/1.0.3.0)
//
// SHOULD WORK WITH (needs hardware testing):
//   - E3DC Multi Connect I Wallbox

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
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/sirupsen/logrus"
	"github.com/spali/go-rscp/rscp"
	"github.com/spf13/cast"
)

// E3dc charger implementation using RSCP protocol.
// Communicates with the E3DC Hauskraftwerk via TCP connection.
type E3dc struct {
	log  *util.Logger // Logger instance for debug/warning output
	conn *rscp.Client // RSCP client connection to E3DC system
	id   uint8        // Wallbox index (0 = first wallbox, 1 = second, etc.)
}

func init() {
	registry.AddCtx("e3dc-rscp", NewE3dcFromConfig)
}

// NewE3dcFromConfig creates an E3DC charger from generic config.
// Called by evcc's charger registry when type "e3dc-rscp" is configured.
//
// Configuration parameters:
//   - uri: IP:Port of E3DC system (default port 5033)
//   - user: E3DC portal username
//   - password: E3DC portal password
//   - key: RSCP encryption key (configured in E3DC Hauskraftwerk settings)
//   - id: Wallbox index (0 = first wallbox)
//   - timeout: Connection timeout (optional)
func NewE3dcFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
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

	// Configure RSCP library logging to use evcc's TRACE level.
	// Setting DebugLevel ensures we get detailed RSCP protocol output,
	// but routing to TRACE.Writer() means it only appears when evcc is in trace mode.
	e3dcOnce.Do(func() {
		rscp.Log.SetLevel(logrus.DebugLevel)
		rscp.Log.SetOutput(log.TRACE.Writer())
	})

	conn, err := rscp.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	wb := &E3dc{
		log:  log,
		conn: conn,
		id:   id,
	}

	// Check wallbox configuration and warn if not optimal for evcc control
	if err := wb.checkConfiguration(); err != nil {
		return nil, err
	}

	return wb, nil
}

// checkConfiguration verifies wallbox settings and adjusts them for evcc control
func (wb *E3dc) checkConfiguration() error {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SUN_MODE_ACTIVE, nil),
		*rscp.NewMessage(rscp.WB_REQ_AUTO_PHASE_SWITCH_ENABLED, nil),
	}))
	if err != nil {
		return fmt.Errorf("failed to query wallbox configuration: %w", err)
	}

	wbData, err := rscpContainer(*res, 3)
	if err != nil {
		return fmt.Errorf("failed to parse wallbox configuration: %v", err)
	}

	// Check and disable sun mode - evcc needs to control charging
	// Note: Sun mode is also checked in ensureSunModeDisabled() on every control command
	// because the user could re-enable it in the E3DC portal at any time
	if sunMode, err := rscpBool(wbData[1]); err == nil && sunMode {
		wb.log.WARN.Println("wallbox sun mode is enabled - disabling for evcc control")
		wb.disableSunMode()
	}

	// Check and disable auto phase switching - evcc needs to control phase switching
	// Note: Auto phase switch is also checked in ensureAutoPhaseDisabled() on phase switch commands
	// because the user could re-enable it in the E3DC portal at any time
	if autoPhase, err := rscpBool(wbData[2]); err == nil && autoPhase {
		wb.log.WARN.Println("wallbox auto phase switching is enabled - disabling for evcc control")
		wb.disableAutoPhaseSwitch()
	}

	// Note: We intentionally do NOT set an initial phase count here.
	// evcc will control phase switching based on charging mode (PV, Min+PV, Fast, etc.).
	// Setting 1 phase on startup would interrupt fast charging (3p) during restarts.

	return nil
}

// disableSunMode sends the command to disable sun mode
func (wb *E3dc) disableSunMode() {
	if _, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SET_SUN_MODE_ACTIVE, false),
	})); err != nil {
		wb.log.ERROR.Printf("failed to disable sun mode: %v", err)
	}
}

// ensureSunModeDisabled checks if sun mode is active and disables it.
// Called before control commands (Enable, MaxCurrent) because the user could
// re-enable sun mode in the E3DC portal at any time without restarting evcc.
func (wb *E3dc) ensureSunModeDisabled() {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SUN_MODE_ACTIVE, nil),
	}))
	if err != nil {
		return
	}

	wbData, err := rscpContainer(*res, 2)
	if err != nil {
		return
	}

	if sunMode, err := rscpBool(wbData[1]); err == nil && sunMode {
		wb.log.WARN.Println("sun mode was re-enabled - disabling for evcc control")
		wb.disableSunMode()
	}
}

// disableAutoPhaseSwitch sends the command to disable automatic phase switching
func (wb *E3dc) disableAutoPhaseSwitch() {
	// Note: WB_REQ_SET_AUTO_PHASE_SWITCH_ENABLED has wrong DataType in go-rscp (None instead of Bool)
	// We must create the message with explicit DataType
	if _, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		{Tag: rscp.WB_REQ_SET_AUTO_PHASE_SWITCH_ENABLED, DataType: rscp.Bool, Value: false},
	})); err != nil {
		wb.log.ERROR.Printf("failed to disable auto phase switch: %v", err)
	}
}

// ensureAutoPhaseDisabled checks if auto phase switching is active and disables it.
// Called before phase switch commands because the user could re-enable it
// in the E3DC portal at any time without restarting evcc.
func (wb *E3dc) ensureAutoPhaseDisabled() {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_AUTO_PHASE_SWITCH_ENABLED, nil),
	}))
	if err != nil {
		return
	}

	wbData, err := rscpContainer(*res, 2)
	if err != nil {
		return
	}

	if autoPhase, err := rscpBool(wbData[1]); err == nil && autoPhase {
		wb.log.WARN.Println("auto phase switch was re-enabled - disabling for evcc control")
		wb.disableAutoPhaseSwitch()
	}
}

// getExternDataAlg retrieves the WB_EXTERN_DATA_ALG status byte array.
// This is the primary source for wallbox status information.
//
// Returns a byte array where:
//   - Byte 0: Unknown
//   - Byte 1: Number of phases (1 or 3)
//   - Byte 2: Status flags (see Status() and Enabled() for bit definitions)
//   - Byte 3: Max charge current in Ampere
//
// Used by Status() and Enabled() to determine charging state.
func (wb *E3dc) getExternDataAlg() ([]byte, error) {
	// RSCP request pattern: WB_REQ_DATA container with WB_INDEX + request tags
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_ALG, nil),
	}))
	if err != nil {
		return nil, err
	}

	// Response structure: WB_DATA[WB_INDEX, WB_EXTERN_DATA_ALG[WB_INDEX, ByteArray]]
	wbData, err := rscpContainer(*res, 2)
	if err != nil {
		return nil, err
	}

	// WB_EXTERN_DATA_ALG is itself a container with index and data
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
	wb.ensureSunModeDisabled()

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

	// WB_EXTERN_DATA_ALG Byte 2 status bits (IEC 61851):
	//   Bit 5 (0b00100000): Charging active  → StatusC
	//   Bit 3 (0b00001000): Vehicle connected → StatusB
	//   Both 0:                               → StatusA
	//
	// Other bits (0,1,2,4,6,7) are additional info (Solar, Abort, etc.)
	// and do not affect the charging state.
	//
	// NOTE: Bit 2 (0b00000100) behavior varies between wallbox models
	// and is NOT used for status detection to ensure compatibility.
	switch {
	case b[2]&0b00100000 != 0: // Bit 5: charging active → StatusC
		return api.StatusC, nil
	case b[2]&0b00001000 != 0: // Bit 3: vehicle connected → StatusB
		return api.StatusB, nil
	default: // Neither Bit 5 nor Bit 3: no vehicle → StatusA
		return api.StatusA, nil
	}
}

// MaxCurrent implements the api.Charger interface.
// Sets the maximum charging current in Ampere (whole numbers only, 6-32A typical range).
func (wb *E3dc) MaxCurrent(current int64) error {
	wb.ensureSunModeDisabled()

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
	var found bool
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
		found = true
		break
	}

	if !found {
		wb.log.WARN.Printf("wallbox index %d not found in DB_TEC_WALLBOX_VALUES - total energy may be inaccurate", wb.id)
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

// powers returns the charging power for each individual phase in watts.
// Used internally by CurrentPower() and Currents().
// Returns (L1, L2, L3) power values - unused phases return 0.
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

	// Response: WB_DATA[WB_INDEX, WB_PM_POWER_L1, WB_PM_POWER_L2, WB_PM_POWER_L3]
	wbData, err := rscpContainer(*res, 4)
	if err != nil {
		return 0, 0, 0, err
	}

	// Extract power values (index 0 is WB_INDEX, 1-3 are the power values)
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

// var _ api.PhaseCurrents = (*E3dc)(nil)

// // Currents implements the api.PhaseCurrents interface
// // Calculates current from power readings as voltage readings are not accessible
// func (wb *E3dc) Currents() (float64, float64, float64, error) {
// 	p1, p2, p3, err := wb.powers()
// 	if err != nil {
// 		return 0, 0, 0, err
// 	}

// 	// Calculate current from power using nominal 230V
// 	// Note: WB_REQ_DIAG_PHASE_VOLTAGE returns ERR_ACCESS_DENIED
// 	const voltage = 230.0
// 	i1 := p1 / voltage
// 	i2 := p2 / voltage
// 	i3 := p3 / voltage

// 	return i1, i2, i3, nil
// }

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

// getSessionData retrieves the session data container from WB_REQ_SESSION.
// Returns all session-related messages (energy, time, RFID, etc.).
// If no vehicle is connected, returns only WB_INDEX with no session data.
func (wb *E3dc) getSessionData() ([]rscp.Message, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_SESSION, nil))
	if err != nil {
		return nil, err
	}

	return rscpContainer(*res, 1)
}

// sessionMessage finds a specific tag in the WB_SESSION response data.
// Used by ChargedEnergy, ChargeDuration, and Identify to extract session values.
// Returns (message, true, nil) if found, (empty, false, nil) if no active session,
// or (empty, false, error) on communication failure.
func (wb *E3dc) sessionMessage(tag rscp.Tag) (rscp.Message, bool, error) {
	sessionData, err := wb.getSessionData()
	if err != nil {
		return rscp.Message{}, false, err
	}

	for _, msg := range sessionData {
		if msg.Tag == tag {
			return msg, true, nil
		}
	}

	return rscp.Message{}, false, nil
}

var _ api.ChargeRater = (*E3dc)(nil)

// ChargedEnergy implements the api.ChargeRater interface
// Returns the energy charged in the current session from WB_SESSION_CHARGED_ENERGY
func (wb *E3dc) ChargedEnergy() (float64, error) {
	msg, found, err := wb.sessionMessage(rscp.WB_SESSION_CHARGED_ENERGY)
	if err != nil || !found {
		return 0, err
	}

	energy, err := rscpFloat64(msg)
	if err != nil {
		return 0, err
	}

	return energy / 1000.0, nil // Wh -> kWh
}

var _ api.ChargeTimer = (*E3dc)(nil)

// ChargeDuration implements the api.ChargeTimer interface
// Returns the active charging duration from WB_SESSION_ACTIVE_CHARGE_TIME
func (wb *E3dc) ChargeDuration() (time.Duration, error) {
	msg, found, err := wb.sessionMessage(rscp.WB_SESSION_ACTIVE_CHARGE_TIME)
	if err != nil || !found {
		return 0, err
	}

	// Session time is in milliseconds (Uint64)
	ms, err := rscpUint64(msg)
	if err != nil {
		return 0, err
	}

	return time.Duration(ms) * time.Millisecond, nil
}

var _ api.Identifier = (*E3dc)(nil)

// Identify implements the api.Identifier interface
// Returns the RFID tag ID from WB_SESSION_AUTH_DATA if a session is active
func (wb *E3dc) Identify() (string, error) {
	msg, found, err := wb.sessionMessage(rscp.WB_SESSION_AUTH_DATA)
	if err != nil || !found {
		return "", err
	}

	return rscpString(msg)
}

var _ api.PhaseSwitcher = (*E3dc)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
// Switches between 1-phase and 3-phase charging
// The wallbox handles the safe switching sequence internally (reduce current, switch, ramp up)
func (wb *E3dc) Phases1p3p(phases int) error {
	if phases != 1 && phases != 3 {
		return fmt.Errorf("invalid phases: %d (must be 1 or 3)", phases)
	}

	wb.ensureSunModeDisabled()
	wb.ensureAutoPhaseDisabled()

	// Perform phase switch
	_, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_SET_NUMBER_PHASES, uint8(phases)),
	}))

	return err
}

var _ api.Diagnosis = (*E3dc)(nil)

// Diagnose implements the api.Diagnosis interface.
// Outputs wallbox information for debugging via evcc's "evcc charger" command.
// Shows device name, firmware, current limits, phase config, and status flags.
func (wb *E3dc) Diagnose() {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_DEVICE_NAME, nil),
		*rscp.NewMessage(rscp.WB_REQ_FIRMWARE_VERSION, nil),
		*rscp.NewMessage(rscp.WB_REQ_MAX_CHARGE_CURRENT, nil),
		*rscp.NewMessage(rscp.WB_REQ_LOWER_CURRENT_LIMIT, nil),
		*rscp.NewMessage(rscp.WB_REQ_UPPER_CURRENT_LIMIT, nil),
		*rscp.NewMessage(rscp.WB_REQ_NUMBER_PHASES, nil),
		*rscp.NewMessage(rscp.WB_REQ_SUN_MODE_ACTIVE, nil),
		*rscp.NewMessage(rscp.WB_REQ_AUTO_PHASE_SWITCH_ENABLED, nil),
		*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_ALG, nil),
	}))
	if err != nil {
		fmt.Printf("\tError: %v\n", err)
		return
	}

	wbData, err := rscpContainer(*res, 10)
	if err != nil {
		fmt.Printf("\tError: %v\n", err)
		return
	}

	if name, err := rscpString(wbData[1]); err == nil {
		fmt.Printf("\tDevice:\t%s\n", name)
	}
	if fw, err := rscpString(wbData[2]); err == nil {
		fmt.Printf("\tFirmware:\t%s\n", fw)
	}
	if current, err := rscpFloat64(wbData[3]); err == nil {
		fmt.Printf("\tMax current:\t%.0fA\n", current)
	}
	if minI, err := rscpFloat64(wbData[4]); err == nil {
		if maxI, err := rscpFloat64(wbData[5]); err == nil {
			fmt.Printf("\tCurrent limits:\t%.0f-%.0fA\n", minI, maxI)
		}
	}
	if phases, err := rscpUint8(wbData[6]); err == nil {
		fmt.Printf("\tPhases:\t%d\n", phases)
	}
	if sunMode, err := rscpBool(wbData[7]); err == nil {
		fmt.Printf("\tSun mode:\t%t\n", sunMode)
	}
	if autoPhase, err := rscpBool(wbData[8]); err == nil {
		fmt.Printf("\tAuto phase switch:\t%t\n", autoPhase)
	}
	if extData, err := rscpContainer(wbData[9], 2); err == nil {
		if b, err := rscpBytes(extData[1]); err == nil && len(b) >= 3 {
			status := b[2]
			var state string
			switch {
			case status&0b00100000 != 0:
				state = "C (charging)"
			case status&0b00001000 != 0:
				state = "B (connected)"
			case status&0b00000100 != 0:
				state = "A (available)"
			default:
				state = "unknown"
			}
			enabled := status&0b01000000 == 0
			fmt.Printf("\tStatus:\t%s\n", state)
			fmt.Printf("\tEnabled:\t%t\n", enabled)
			fmt.Printf("\tStatus bits:\t%08b\n", status)
		}
	}
}

// ===========================================================================
// RSCP Helper Functions
// ===========================================================================
// These functions handle the parsing of RSCP protocol responses.
// RSCP messages contain typed values that need to be extracted and validated.
//
// Typical usage pattern:
//   1. Send request via wb.conn.Send()
//   2. Parse response container via rscpContainer()
//   3. Extract typed values via rscpFloat64(), rscpBool(), rscpString(), etc.
// ===========================================================================

// rscpError extracts error messages from RSCP responses.
// RSCP uses a special Error datatype to indicate failures (e.g., ERR_ACCESS_DENIED).
func rscpError(msg ...rscp.Message) error {
	var errs []error
	for _, m := range msg {
		if m.DataType == rscp.Error {
			errs = append(errs, errors.New(rscp.RscpError(cast.ToUint32(m.Value)).String()))
		}
	}
	return errors.Join(errs...)
}

// rscpContainer extracts and validates a container message.
// RSCP containers hold multiple sub-messages (like WB_DATA holding WB_INDEX + values).
// The length parameter specifies minimum expected sub-messages.
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

// rscpBytes extracts a byte array from an RSCP message.
// Used for WB_EXTERN_DATA_ALG which contains status flags as raw bytes.
func rscpBytes(msg rscp.Message) ([]byte, error) {
	return rscpValue(msg, func(data any) ([]byte, error) {
		b, ok := data.([]uint8)
		if !ok {
			return nil, errors.New("invalid response")
		}
		return b, nil
	})
}

// rscpFloat64 extracts a float64 value from an RSCP message.
// Used for power (W), energy (Wh), and current (A) values.
// Handles automatic type conversion from RSCP's various numeric types.
func rscpFloat64(msg rscp.Message) (float64, error) {
	return rscpValue(msg, func(data any) (float64, error) {
		return cast.ToFloat64E(data)
	})
}

// rscpUint8 extracts a uint8 value from an RSCP message.
// Used for WB_INDEX, WB_NUMBER_PHASES, and similar small integer values.
func rscpUint8(msg rscp.Message) (uint8, error) {
	return rscpValue(msg, func(data any) (uint8, error) {
		return cast.ToUint8E(data)
	})
}

// rscpString extracts a string value from an RSCP message.
// Used for WB_DEVICE_NAME, WB_FIRMWARE_VERSION, WB_SESSION_AUTH_DATA (RFID), etc.
func rscpString(msg rscp.Message) (string, error) {
	return rscpValue(msg, func(data any) (string, error) {
		return cast.ToStringE(data)
	})
}

// rscpBool extracts a bool value from an RSCP message.
// Used for WB_SUN_MODE_ACTIVE, WB_AUTO_PHASE_SWITCH_ENABLED, etc.
func rscpBool(msg rscp.Message) (bool, error) {
	return rscpValue(msg, func(data any) (bool, error) {
		b, ok := data.(bool)
		if !ok {
			return false, errors.New("invalid response")
		}
		return b, nil
	})
}

// rscpUint64 extracts a uint64 value from an RSCP message.
// Used for WB_SESSION_ACTIVE_CHARGE_TIME (milliseconds), etc.
func rscpUint64(msg rscp.Message) (uint64, error) {
	return rscpValue(msg, func(data any) (uint64, error) {
		return cast.ToUint64E(data)
	})
}

// rscpValue is a generic helper for extracting typed values from RSCP messages.
// Takes a conversion function that transforms the raw value to the desired type.
// First checks for RSCP errors, then applies the conversion function.
func rscpValue[T any](msg rscp.Message, fun func(any) (T, error)) (T, error) {
	var zero T
	if err := rscpError(msg); err != nil {
		return zero, err
	}

	return fun(msg.Value)
}
