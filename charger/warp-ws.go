package charger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/charger/warp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type wsRole int

const (
	wsRoleMain wsRole = iota
	wsRolePM
)

type WarpWS struct {
	*warp.Connection
	implement.Caps
	pm *warp.Connection // separate Energy Manager

	// config
	log        *util.Logger
	meterIndex uint

	mu sync.RWMutex

	// capabilities
	features []string

	// evse
	evse       warp.Evse
	maxCurrent int64 // input from evcc

	// meter
	// warp.MvidValues() is the list of meter value IDs that are of interest for EVCC.
	// indices maps those IDs to the index in the meters/X/values API.
	indices map[warp.Mvid]int
	values  map[warp.Mvid]float64

	// nfc
	chargeTracker warp.ChargeTrackerCurrentCharge

	// ev (WARP4, ISO 15118)
	evState *warp.EvState

	// power manager
	pmState          *warp.PmState
	pmLowLevelState  *warp.PmLowLevelState
	lastPhasesWanted int // 0=never set; 1 or 3
}

func init() {
	registry.AddCtx("warp-ws", NewWarpWSFromConfig)
}

func NewWarpWSFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc struct {
		URI                   string
		User                  string
		Password              string
		EnergyManagerURI      string
		EnergyManagerUser     string
		EnergyManagerPassword string
		EnergyMeterIndex      uint

		DisablePhaseAutoSwitch_ bool `mapstructure:"disablePhaseAutoSwitch"` // TODO deprecated
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	w, err := NewWarpWS(ctx, cc.URI, cc.User, cc.Password, cc.EnergyManagerURI, cc.EnergyManagerUser, cc.EnergyManagerPassword, cc.EnergyMeterIndex)
	if err != nil {
		return nil, err
	}

	// If the energy manager URI is set, use the energy manager for phase switching.
	if cc.EnergyManagerURI != "" {
		implement.Has(w, implement.PhaseSwitcher(w.phases1p3p))
		implement.Has(w, implement.PhaseGetter(w.getPhases))
	}

	return w, nil
}

func NewWarpWS(ctx context.Context, uri, user, pass, emURI, emUser, emPass string, meterIndex uint) (*WarpWS, error) {
	log := util.NewLogger("warp-ws")

	w := &WarpWS{
		Connection: warp.NewConnection(log, uri, user, pass),
		Caps:       implement.New(),
		log:        log,
		meterIndex: meterIndex,
		indices:    make(map[warp.Mvid]int, len(warp.MvidValues())),
		values:     make(map[warp.Mvid]float64, len(warp.MvidValues())),
	}

	if emURI != "" {
		w.pm = warp.NewConnection(log, emURI, emUser, emPass)
	} else {
		w.pm = w.Connection
	}

	wsURI, err := parseURI(w.URI)
	if err != nil {
		return nil, err
	}

	go w.run(ctx, wsRoleMain, w.Connection.Client, wsURI)
	if emURI != "" {
		pmWsURI, err := parseURI(w.pm.URI)
		if err != nil {
			return nil, err
		}
		go w.run(ctx, wsRolePM, w.pm.Client, pmWsURI)
	}

	return w, nil
}

func (w *WarpWS) run(ctx context.Context, role wsRole, client *http.Client, wsURI string) {
	bo := backoff.NewExponentialBackOff(
		backoff.WithMaxElapsedTime(0),
		backoff.WithMaxInterval(30*time.Second),
	)

	for ctx.Err() == nil {
		w.log.DEBUG.Println("websocket: connecting")

		conn, _, err := websocket.Dial(ctx, wsURI, &websocket.DialOptions{HTTPClient: client})
		if err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				w.log.ERROR.Printf("websocket: %v", err)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(bo.NextBackOff()):
			}

			continue
		}

		bo.Reset()

		if role == wsRolePM {
			if err := w.resendLastPhasesWantedIfAny(); err != nil {
				w.log.WARN.Printf("resend phases_wanted on reconnect: %v", err)
			}
		}

		if err := w.handleConnection(ctx, role, conn); err != nil {
			w.log.ERROR.Println(err)
		}
	}
}

func (w *WarpWS) resendLastPhasesWantedIfAny() error {
	w.mu.RLock()
	phases := w.lastPhasesWanted
	w.mu.RUnlock()

	if phases == 0 {
		return nil
	}

	return w.postPhasesWanted(phases)
}

// Returns parsed URI and hostname
func parseURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = path.Join(u.Path, "/ws")

	return u.String(), nil
}

func isPmTopic(topic string) bool {
	return strings.HasPrefix(topic, "power_manager/")
}

func (w *WarpWS) handleConnection(ctx context.Context, role wsRole, conn *websocket.Conn) error {
	defer conn.Close(websocket.StatusInternalError, "reconnect")
	for {
		msgType, r, err := conn.Reader(ctx)
		if err != nil {
			return err
		}
		if msgType != websocket.MessageText {
			continue // next frame
		}

		dec := json.NewDecoder(r)
		for {
			var event struct {
				Topic   string          `json:"topic"`
				Payload json.RawMessage `json:"payload"`
			}
			if err := dec.Decode(&event); err != nil {
				if errors.Is(err, io.EOF) {
					break //next frame
				}
				return err
			}

			// only drop PM topics on the main WS when a dedicated PM connection exists;
			// on single-WS setups (WARP3) PM events arrive here and must be processed
			if role == wsRoleMain && w.pm != w.Connection && isPmTopic(event.Topic) {
				continue
			}

			w.log.TRACE.Printf("websocket: event %s: %s", event.Topic, event.Payload)
			if err := w.handleEvent(event.Topic, event.Payload); err != nil {
				w.log.ERROR.Printf("bad payload for topic %s: %v", event.Topic, err)
			}
		}
	}
}

func (w *WarpWS) handleEvent(topic string, payload json.RawMessage) error {
	metersValueIDsTopic := fmt.Sprintf("meters/%d/value_ids", w.meterIndex)
	metersValuesTopic := fmt.Sprintf("meters/%d/values", w.meterIndex)

	w.mu.Lock()
	defer w.mu.Unlock()

	var err error
	switch topic {
	case "charge_tracker/current_charge":
		err = json.Unmarshal(payload, &w.chargeTracker)
	case "ev/state":
		err = json.Unmarshal(payload, &w.evState)
	case "evse/external_current":
		err = json.Unmarshal(payload, &w.evse.ExternalCurrent)
	case "evse/user_current":
		err = json.Unmarshal(payload, &w.evse.UserCurrent)
	case "evse/user_enabled":
		err = json.Unmarshal(payload, &w.evse.UserEnabled)
	case "evse/state":
		err = json.Unmarshal(payload, &w.evse.State)
	case "evse/low_level_state":
		err = json.Unmarshal(payload, &w.evse.LowLevelState)
	case "evse/slots":
		err = json.Unmarshal(payload, &w.evse.Slots)
	case "evse/phase_auto_switch":
		// Phase Auto Switching needs to be disabled WARP2 or newer
		// Necessary if charging 1p only vehicles
		var auto_switch warp.EvsePhaseAutoSwitch
		if err = json.Unmarshal(payload, &auto_switch); err != nil {
			return err
		}

		if auto_switch.Enabled {
			// A bit ugly, but we don't want to hold the mutex while sending an HTTP request.
			w.mu.Unlock()
			err = w.disablePhaseAutoSwitch()
			w.mu.Lock()
			if err == nil {
				w.log.WARN.Println("disabled WARP phase auto switching")
			}
		}
	case "info/features":
		err = json.Unmarshal(payload, &w.features)

		// Feature: ISO 15118 (WARP4): vehicle soc and mac exposed via ev/state
		hasIso15118 := w.hasFeature(warp.FeatureIso15118)
		if hasIso15118 {
			implement.Has(w, implement.Battery(w.soc))
			implement.Has(w, implement.BatteryCapacity(w.capacity))
		}

		// Feature: NFC
		if w.hasFeature(warp.FeatureNfc) || hasIso15118 {
			implement.Has(w, implement.Identifier(w.identify))
		}

		if w.hasFeature(warp.FeaturePhaseSwitch) {
			implement.Has(w, implement.PhaseSwitcher(w.phases1p3p))
			implement.Has(w, implement.PhaseGetter(w.getPhases))
		}

	case metersValueIDsTopic:
		var ids []warp.Mvid
		if err = json.Unmarshal(payload, &ids); err != nil {
			return err
		}

		for _, needle := range warp.MvidValues() {
			w.indices[needle] = -1

			for idx, hay := range ids {
				if hay == needle {
					w.indices[needle] = idx
					break
				}
			}
		}

		if w.indices[warp.MvidPower] != -1 {
			implement.Has(w, implement.Meter(w.currentPower))
		}

		if w.indices[warp.MvidEnergy] != -1 {
			implement.Has(w, implement.MeterEnergy(w.totalEnergy))
			implement.Has(w, implement.ChargeRater(w.chargedEnergy))
		}

		if w.indices[warp.MvidCurrentL1] != -1 && w.indices[warp.MvidCurrentL2] != -1 && w.indices[warp.MvidCurrentL3] != -1 {
			implement.Has(w, implement.PhaseCurrents(w.currents))
		}

		if w.indices[warp.MvidVoltageL1] != -1 && w.indices[warp.MvidVoltageL2] != -1 && w.indices[warp.MvidVoltageL3] != -1 {
			implement.Has(w, implement.PhaseVoltages(w.voltages))
		}

		if w.indices[warp.MvidPowerL1] != -1 && w.indices[warp.MvidPowerL2] != -1 && w.indices[warp.MvidPowerL3] != -1 {
			implement.Has(w, implement.PhasePowers(w.powers))
		}
	case metersValuesTopic:
		var values []warp.FloatWithNaN
		if err := json.Unmarshal(payload, &values); err != nil {
			return err
		}

		for mvid, idx := range w.indices {
			if idx >= 0 && idx < len(values) {
				w.values[mvid] = float64(values[idx])
			} else {
				w.values[mvid] = math.NaN()
			}
		}
	case "power_manager/state":
		err = json.Unmarshal(payload, &w.pmState)
	case "power_manager/low_level_state":
		err = json.Unmarshal(payload, &w.pmLowLevelState)
	}
	return err
}

func (w *WarpWS) hasFeature(feature string) bool {
	return slices.Contains(w.features, feature)
}

func (w *WarpWS) Enable(enable bool) error {
	var curr int64
	if enable {
		curr = w.maxCurrent
	}
	return w.setCurrent(curr)
}

func (w *WarpWS) Enabled() (bool, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.evse.ExternalCurrent.Current > 0, nil
}

// MaxCurrent implements the api.Charger interface
func (w *WarpWS) MaxCurrent(current int64) error {
	return w.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*WarpWS)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (w *WarpWS) MaxCurrentMillis(current float64) error {
	curr := int64(current * 1e3)
	err := w.setCurrent(curr)
	if err == nil {
		w.maxCurrent = curr
	}
	return err
}

func (w *WarpWS) statusFromEvseStatus(state int) (api.ChargeStatus, error) {
	if state < 3 {
		return []api.ChargeStatus{api.StatusA, api.StatusB, api.StatusC}[state], nil
	}
	return api.StatusNone, fmt.Errorf("unknown evse status: %d", state)
}

func (w *WarpWS) Status() (api.ChargeStatus, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.statusFromEvseStatus(w.evse.State.Iec61851State)
}

func (w *WarpWS) StatusReason() (api.Reason, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if status, err := w.statusFromEvseStatus(w.evse.State.Iec61851State); err != nil {
		return api.ReasonUnknown, err
	} else if status == api.StatusB && w.evse.UserEnabled.Enabled && w.evse.UserCurrent.Current == 0 {
		return api.ReasonWaitingForAuthorization, nil
	}
	return api.ReasonUnknown, nil
}

func (w *WarpWS) currentPower() (float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if math.IsNaN(w.values[warp.MvidPower]) {
		return 0, api.ErrNotAvailable
	}

	return w.values[warp.MvidPower], nil
}

func (w *WarpWS) totalEnergy() (float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if math.IsNaN(w.values[warp.MvidEnergy]) {
		return 0, api.ErrNotAvailable
	}

	return w.values[warp.MvidEnergy], nil
}

func (w *WarpWS) currents() (float64, float64, float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if math.IsNaN(w.values[warp.MvidCurrentL1]) ||
		math.IsNaN(w.values[warp.MvidCurrentL2]) ||
		math.IsNaN(w.values[warp.MvidCurrentL3]) {
		return 0, 0, 0, api.ErrNotAvailable
	}

	return w.values[warp.MvidCurrentL1], w.values[warp.MvidCurrentL2], w.values[warp.MvidCurrentL3], nil
}

func (w *WarpWS) voltages() (float64, float64, float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if math.IsNaN(w.values[warp.MvidVoltageL1]) ||
		math.IsNaN(w.values[warp.MvidVoltageL2]) ||
		math.IsNaN(w.values[warp.MvidVoltageL3]) {
		return 0, 0, 0, api.ErrNotAvailable
	}

	return w.values[warp.MvidVoltageL1], w.values[warp.MvidVoltageL2], w.values[warp.MvidVoltageL3], nil
}

func (w *WarpWS) powers() (float64, float64, float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if math.IsNaN(w.values[warp.MvidPowerL1]) ||
		math.IsNaN(w.values[warp.MvidPowerL2]) ||
		math.IsNaN(w.values[warp.MvidPowerL3]) {
		return 0, 0, 0, api.ErrNotAvailable
	}

	return w.values[warp.MvidPowerL1], w.values[warp.MvidPowerL2], w.values[warp.MvidPowerL3], nil
}

// identify reports the vehicle mac read via ISO 15118 before the RFID tag
func (w *WarpWS) identify() ([]string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var ids []string
	if w.evState != nil && w.evState.Mac != "" {
		ids = append(ids, w.evState.Mac)
	}
	if tag := w.chargeTracker.AuthorizationInfo.TagId; tag != "" {
		ids = append(ids, tag)
	}

	return ids, nil
}

// soc implements the api.Battery interface
func (w *WarpWS) soc() (float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.evState != nil && w.evState.Soc != nil {
		return *w.evState.Soc, nil
	}
	return 0, api.ErrNotAvailable
}

// capacity implements the api.BatteryCapacity interface
func (w *WarpWS) capacity() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.evState != nil && w.evState.Capacity != nil {
		return *w.evState.Capacity
	}
	return 0
}

func (w *WarpWS) chargedEnergy() (float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.chargeTracker.UserId == -1 {
		// Currently not charging.
		// TODO: is this an error or shall we return 0, nil here?
		return 0, api.ErrNotAvailable
	}

	var start = float64(w.chargeTracker.MeterStart)
	var now = w.values[warp.MvidEnergy]

	if math.IsNaN(start) || math.IsNaN(now) {
		return 0, api.ErrNotAvailable
	}

	return now - start, nil
}

// ChargeDuration implements the api.ChargeTimer interface
func (w *WarpWS) ChargeDuration() (time.Duration, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Time-keeping is hard.
	// Older WARP firmwares supported only <= 32 bit datatypes on the API.
	// Thus the start unix timestamp is an int32 in *minutes*.
	// Also if the wall-clock time was unknown when the charge started, the timestamp is 0.
	// (For example when the NTP sync failed)
	//
	// We can rely on the EVSE uptime instead, but this also is a 32 bit int in milliseconds.
	// It will overflow after ~ 50 days.
	// Also the EVSE clock does drift a bit.
	//
	// If the wall-clock time at charge start is unknown, we don't report anything.
	// If the charge start (based on the wall-clock time) was more than two hours ago, report the wall-clock duration.
	//   In that case it's probably fine that we jump up to 59 seconds.
	// If it was less than two hours ago, use the EVSE's clock.

	if w.chargeTracker.UserId == -1 {
		// Currently not charging.
		// TODO: is this an error or shall we return 0, nil here?
		return 0, api.ErrNotAvailable
	}

	if w.chargeTracker.TimestampMinutes == 0 {
		return 0, api.ErrNotAvailable
	}
	var wallclock_duration = time.Now().Sub(time.Unix(int64(w.chargeTracker.TimestampMinutes)*60, 0))

	if wallclock_duration.Hours() > 2 {
		return wallclock_duration, nil
	}

	var evse_duration_ms uint32

	if w.evse.LowLevelState.Uptime >= w.chargeTracker.EvseUptimeStart {
		evse_duration_ms = w.evse.LowLevelState.Uptime - w.chargeTracker.EvseUptimeStart
	} else {
		evse_duration_ms = w.chargeTracker.EvseUptimeStart - w.evse.LowLevelState.Uptime
	}
	return time.Duration(int64(evse_duration_ms) * 1000 * 1000), nil
}

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (w *WarpWS) GetMinMaxCurrent() (float64, float64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var maxCurrent = min(w.evse.Slots[0].MaxCurrent, w.evse.Slots[1].MaxCurrent)
	return 6, float64(maxCurrent) / 1000, nil
}

func (w *WarpWS) setCurrent(curr int64) error {
	uri := fmt.Sprintf("%s/evse/external_current", w.URI)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(map[string]int64{"current": curr}), request.JSONEncoding)
	_, err := w.Do(req)
	return err
}

func (w *WarpWS) disablePhaseAutoSwitch() error {
	uri := fmt.Sprintf("%s/evse/phase_auto_switch", w.URI)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(map[string]bool{"enabled": false}), request.JSONEncoding)
	_, err := w.Do(req)
	return err
}

func (w *WarpWS) postPhasesWanted(phases int) error {
	uri := fmt.Sprintf("%s/power_manager/external_control", w.pm.URI)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(map[string]int{"phases_wanted": phases}), request.JSONEncoding)
	_, err := w.pm.Do(req)
	return err
}

// phases1p3p implements the api.PhaseSwitcher interface
func (w *WarpWS) phases1p3p(phases int) error {
	// ExternalControlDeactivated is the WEM/WARP3 idle state before any
	// phases_wanted has been sent — the POST below activates external control.
	// Only block on states the POST cannot resolve.
	ec, err := w.ensurePmState()
	if err != nil {
		return err
	}
	if ec.ExternalControl == warp.ExternalControlRuntimeConditionsNotMet ||
		ec.ExternalControl == warp.ExternalControlCurrentlySwitching {
		return fmt.Errorf("external control %v: %w", ec.ExternalControl, api.ErrNotAvailable)
	}

	if err := w.postPhasesWanted(phases); err != nil {
		return err
	}
	w.mu.Lock()
	w.lastPhasesWanted = phases
	w.mu.Unlock()
	return nil
}

// getPhases implements the api.PhaseGetter interface
func (w *WarpWS) getPhases() (int, error) {
	s, err := w.ensurePmLowLevelState()
	if err != nil {
		return 0, err
	}
	if s.Is3phase {
		return 3, nil
	}
	return 1, nil
}

func (w *WarpWS) ensurePmLowLevelState() (warp.PmLowLevelState, error) {
	w.mu.RLock()
	s := w.pmLowLevelState
	w.mu.RUnlock()
	if s != nil {
		return *s, nil
	}

	var ns warp.PmLowLevelState
	if err := w.pm.GetJSON(fmt.Sprintf("%s/power_manager/low_level_state", w.pm.URI), &ns); err != nil {
		return warp.PmLowLevelState{}, err
	}

	w.mu.Lock()
	w.pmLowLevelState = &ns
	w.mu.Unlock()
	return ns, nil
}

func (w *WarpWS) ensurePmState() (warp.PmState, error) {
	w.mu.RLock()
	s := w.pmState
	w.mu.RUnlock()
	if s != nil {
		return *s, nil
	}

	var res warp.PmState
	if err := w.pm.GetJSON(fmt.Sprintf("%s/power_manager/state", w.pm.URI), &res); err != nil {
		return warp.PmState{}, err
	}

	w.mu.Lock()
	w.pmState = &res
	w.mu.Unlock()
	return res, nil
}
