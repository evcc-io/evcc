package charger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	meter                    warp.MeterValues
	meterMap                 map[int]int
	hasCurrents, hasVoltages bool // meter actually reports per-phase currents/voltages

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

	// Feature: Meter -> Meter is legacy API, Meters is the new API
	if w.hasFeature(warp.FeatureMeter) || w.hasFeature(warp.FeatureMeters) {
		implement.Has(w, implement.Meter(w.currentPower))
		implement.Has(w, implement.MeterEnergy(w.totalEnergy))
	}

	// Feature: Meters | MeterAllValues
	if w.hasFeature(warp.FeatureMeters) || w.hasFeature(warp.FeatureMeterAllValues) {
		implement.Has(w, implement.PhaseCurrents(w.currents))
		implement.Has(w, implement.PhaseVoltages(w.voltages))
	}

	// Feature: ISO 15118 (WARP4): vehicle soc and mac exposed via ev/state
	hasIso15118 := w.hasFeature(warp.FeatureIso15118)
	if hasIso15118 {
		implement.Has(w, implement.Battery(w.soc))
	}

	// Feature: NFC
	if w.hasFeature(warp.FeatureNfc) || hasIso15118 {
		implement.Has(w, implement.Identifier(w.identify))
	}

	// Feature: Phase Switching
	// only setup phase switching methods if power manager endpoint is set
	if (w.hasFeature(warp.FeaturePhaseSwitch) || cc.EnergyManagerURI != "") && w.pm != nil {
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
		meterMap:   map[int]int{},
	}

	if err := w.GetJSON(fmt.Sprintf("%s/info/features", w.URI), &w.features); err != nil {
		return nil, err
	}

	if emURI != "" {
		w.pm = warp.NewConnection(log, emURI, emUser, emPass)
	} else {
		w.pm = w.Connection
	}

	// Phase Auto Switching needs to be disabled for WARP3 and WARP2 + EM
	// Necessary if charging 1p only vehicles
	typ, err := w.getWarpType()
	if err != nil {
		return nil, err
	}
	if typ == "warp3" || typ == "warp4" || (typ == "warp2" && emURI != "") {
		enabled, err := w.disablePhaseAutoSwitch()
		if err != nil {
			return nil, err
		}
		if enabled {
			w.log.WARN.Println("disabled WARP phase auto switching")
		}
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

	u.Scheme = "ws"
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
	case "meter/all_values":
		if !w.hasFeature(warp.FeatureMeterAllValues) || w.hasFeature(warp.FeatureMeters) {
			return nil
		}
		var values []float64
		if err = json.Unmarshal(payload, &values); err == nil && len(values) > 5 {
			copy(w.meter.Voltages[:], values[:3])
			copy(w.meter.Currents[:], values[3:6])
			w.hasVoltages, w.hasCurrents = true, true
		}
	case "meter/values":
		if !w.hasFeature(warp.FeatureMeter) || w.hasFeature(warp.FeatureMeters) {
			return nil
		}
		err = json.Unmarshal(payload, &w.meter)
	case metersValueIDsTopic:
		var ids []int
		if err = json.Unmarshal(payload, &ids); err != nil {
			return err
		}
		w.meterMap = make(map[int]int, len(ids))
		for i, id := range ids {
			w.meterMap[id] = i
		}
	case metersValuesTopic:
		var values []float64
		if err := json.Unmarshal(payload, &values); err != nil {
			return err
		}

		get := func(id int) (float64, bool) {
			if idx, ok := w.meterMap[id]; ok && idx < len(values) {
				return values[idx], true
			}
			return 0, false
		}

		s := warp.DefaultSchema
		if v, ok := get(s.PowerID); ok {
			w.meter.Power = v
		}
		if v, ok := get(s.EnergyAbsID); ok {
			w.meter.EnergyAbs = v
		}
		for p, ids := range s.Phases {
			if v, ok := get(ids.CurrentID); ok {
				w.meter.Currents[p] = v
				w.hasCurrents = true
			}
			if v, ok := get(ids.VoltageID); ok {
				w.meter.Voltages[p] = v
				w.hasVoltages = true
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
	return w.meter.Power, nil
}

func (w *WarpWS) totalEnergy() (float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.meter.EnergyAbs, nil
}

func (w *WarpWS) currents() (float64, float64, float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if !w.hasCurrents {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return w.meter.Currents[0], w.meter.Currents[1], w.meter.Currents[2], nil
}

func (w *WarpWS) voltages() (float64, float64, float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if !w.hasVoltages {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return w.meter.Voltages[0], w.meter.Voltages[1], w.meter.Voltages[2], nil
}

// identify prefers the vehicle mac read via ISO 15118 over the RFID tag
func (w *WarpWS) identify() (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.evState != nil && w.evState.Mac != "" {
		return w.evState.Mac, nil
	}
	return w.chargeTracker.AuthorizationInfo.TagId, nil
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

func (w *WarpWS) setCurrent(curr int64) error {
	uri := fmt.Sprintf("%s/evse/external_current", w.URI)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(map[string]int64{"current": curr}), request.JSONEncoding)
	_, err := w.Do(req)
	return err
}

func (w *WarpWS) disablePhaseAutoSwitch() (bool, error) {
	uri := fmt.Sprintf("%s/evse/phase_auto_switch", w.URI)
	var state struct {
		Enabled bool `json:"enabled"`
	}
	if err := w.GetJSON(uri, &state); err != nil {
		return false, err
	}
	if !state.Enabled {
		return false, nil
	}
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(map[string]bool{"enabled": false}), request.JSONEncoding)
	_, err := w.Do(req)
	return true, err
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

func (w *WarpWS) getWarpType() (string, error) {
	var res warp.Name
	uri := fmt.Sprintf("%s/info/name", w.URI)
	err := w.GetJSON(uri, &res)
	return res.WarpType, err
}
