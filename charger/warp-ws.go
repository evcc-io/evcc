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
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/warp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type WarpWS struct {
	*warp.Connection
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
	meter    warp.MeterValues
	meterMap map[int]int

	// nfc
	chargeTracker warp.ChargeTrackerCurrentCharge

	// power manager
	pmState         *warp.PmState
	pmLowLevelState *warp.PmLowLevelState
}

func init() {
	registry.AddCtx("warp-ws", NewWarpWSFromConfig)
}

//go:generate go tool decorate -f decorateWarpWS -b *WarpWS -r api.Charger -t api.Meter,api.MeterEnergy,api.PhaseCurrents,api.PhaseVoltages,api.Identifier,api.PhaseSwitcher,api.PhaseGetter

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
	var currentPower, totalEnergy func() (float64, error)
	if w.hasFeature(warp.FeatureMeter) || w.hasFeature(warp.FeatureMeters) {
		currentPower = w.currentPower
		totalEnergy = w.totalEnergy
	}

	// Feature: Meters | MeterAllValues
	var currents, voltages func() (float64, float64, float64, error)
	if w.hasFeature(warp.FeatureMeters) || w.hasFeature(warp.FeatureMeterAllValues) {
		currents = w.currents
		voltages = w.voltages
	}

	// Feature: NFC
	var identify func() (string, error)
	if w.hasFeature(warp.FeatureNfc) {
		identify = w.identify
	}

	// Feature: Phase Switching
	// only setup phase switching methods if power manager endpoint is set
	var phases func(int) error
	var getPhases func() (int, error)
	if (w.hasFeature(warp.FeaturePhaseSwitch) || cc.EnergyManagerURI != "") && w.pm != nil {
		if res, err := w.ensurePmState(); err == nil && res.ExternalControl != warp.ExternalControlDeactivated {
			w.pmState = &res
			phases = w.phases1p3p
			getPhases = w.getPhases
		}
	}

	// Phase Auto Switching needs to be disabled for WARP3 and WARP2 + EM
	// Necessary if charging 1p only vehicles
	typ, err := w.getWarpType()
	if err != nil {
		return nil, err
	}
	if typ == "warp3" || (typ == "warp2" && w.pm != nil && w.pm != w.Connection) {
		if err := w.disablePhaseAutoSwitch(); err != nil {
			return nil, err
		}
		w.log.TRACE.Println("disabled phase auto switching")
	}

	return decorateWarpWS(w, currentPower, totalEnergy, currents, voltages, identify, phases, getPhases), nil
}

func NewWarpWS(ctx context.Context, uri, user, pass, emURI, emUser, emPass string, meterIndex uint) (*WarpWS, error) {
	log := util.NewLogger("warp-ws")

	w := &WarpWS{
		Connection: warp.NewConnection(log, uri, user, pass),
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

	wsURI, err := parseURI(w.URI)
	if err != nil {
		return nil, err
	}

	go w.run(ctx, wsURI)

	return w, nil
}

func (w *WarpWS) run(ctx context.Context, wsURI string) {
	bo := backoff.NewExponentialBackOff(
		backoff.WithMaxElapsedTime(0),
		backoff.WithMaxInterval(30*time.Second),
	)

	for ctx.Err() == nil {
		w.log.DEBUG.Println("websocket: connecting")

		conn, _, err := websocket.Dial(ctx, wsURI, &websocket.DialOptions{HTTPClient: w.Client})
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

		if err := w.handleConnection(ctx, conn); err != nil {
			w.log.ERROR.Println(err)
		}
	}
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

func (w *WarpWS) handleConnection(ctx context.Context, conn *websocket.Conn) error {
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
		err = json.Unmarshal(payload, &w.meter.TmpValues)
		if len(w.meter.TmpValues) > 5 {
			copy(w.meter.Voltages[:], w.meter.TmpValues[:3])
			copy(w.meter.Currents[:], w.meter.TmpValues[3:6])
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
		if err := json.Unmarshal(payload, &w.meter.TmpValues); err != nil {
			return err
		}

		get := func(id int) (float64, bool) {
			if idx, ok := w.meterMap[id]; ok && idx < len(w.meter.TmpValues) {
				return w.meter.TmpValues[idx], true
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
			}
			if v, ok := get(ids.VoltageID); ok {
				w.meter.Voltages[p] = v
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
	return w.meter.Currents[0], w.meter.Currents[1], w.meter.Currents[2], nil
}

func (w *WarpWS) voltages() (float64, float64, float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.meter.Voltages[0], w.meter.Voltages[1], w.meter.Voltages[2], nil
}

func (w *WarpWS) identify() (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.chargeTracker.AuthorizationInfo.TagId, nil
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

// phases1p3p implements the api.PhaseSwitcher interface
func (w *WarpWS) phases1p3p(phases int) error {
	// ensure that phases can be switched
	if ec, err := w.ensurePmState(); err != nil || ec.ExternalControl > warp.ExternalControlAvailable {
		return fmt.Errorf("external control not available: %d", ec.ExternalControl)
	}

	uri := fmt.Sprintf("%s/power_manager/external_control", w.pm.URI)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(map[string]int{"phases_wanted": phases}), request.JSONEncoding)
	_, err := w.pm.Do(req)
	return err
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

	return res, nil
}

func (w *WarpWS) getWarpType() (string, error) {
	var res warp.Name
	uri := fmt.Sprintf("%s/info/name", w.URI)
	err := w.GetJSON(uri, &res)
	return res.WarpType, err
}
