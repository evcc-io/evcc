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
	"github.com/evcc-io/evcc/charger/warp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

type WarpWS struct {
	*request.Helper

	// config
	pmHelper   *request.Helper
	log        *util.Logger
	uri        string
	pmURI      string
	meterIndex uint

	mu sync.RWMutex

	// capabilities
	features []string

	// evse
	evse       warp.Evse
	maxCurrent int64 // input from evcc

	// meter
	meter               warp.MeterValues
	meterMap            map[int]int
	metersValueIDsTopic string
	metersValuesTopic   string

	// nfc
	chargeTracker warp.ChargeTrackerCurrentCharge

	// power manager
	pmState         warp.PmState
	pmLowLevelState warp.PmLowLevelState
}

type warpEvent struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

var _ api.ChargerEx = (*WarpWS)(nil)

func init() {
	registry.AddCtx("warp-ws", NewWarpWSFromConfig)
}

//go:generate go tool decorate -f decorateWarpWS -b *WarpWS -r api.Charger -t api.Meter,api.MeterEnergy,api.PhaseCurrents,api.PhaseVoltages,api.Identifier,api.PhaseSwitcher,api.PhaseGetter

func NewWarpWSFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc struct {
		URI                    string
		User                   string
		Password               string
		EnergyManagerURI       string
		EnergyManagerUser      string
		EnergyManagerPassword  string
		DisablePhaseAutoSwitch bool
		EnergyMeterIndex       uint
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewWarpWS(ctx, cc.URI, cc.User, cc.Password, cc.EnergyMeterIndex)
	if err != nil {
		return nil, err
	}

	// Feature: Meter -> Meter is legacy API, Meters is the new API
	var currentPower, totalEnergy func() (float64, error)
	if wb.hasFeature(warp.FeatureMeter) || wb.hasFeature(warp.FeatureMeters) {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
	}

	// Feature: Meters | MeterAllValues
	var currents, voltages func() (float64, float64, float64, error)
	if wb.hasFeature(warp.FeatureMeters) || wb.hasFeature(warp.FeatureMeterAllValues) {
		currents = wb.currents
		voltages = wb.voltages
	}

	// Feature: Phase Switching
	if wb.hasFeature(warp.FeaturePhaseSwitch) && wb.pmURI == "" {
		wb.pmURI = wb.uri
		wb.pmHelper = wb.Helper
	} else if cc.EnergyManagerURI != "" { // fallback to Energy Manager
		wb.pmURI, err = parseURI(cc.EnergyManagerURI, false)
		if wb.pmURI == "" {
			return nil, err
		} else if err != nil {
			wb.log.DEBUG.Println(err)
		}
		wb.pmHelper = request.NewHelper(wb.log)
		if cc.EnergyManagerUser != "" {
			wb.pmHelper.Client.Transport = digest.NewTransport(cc.EnergyManagerUser, cc.EnergyManagerPassword, wb.pmHelper.Client.Transport)
		}
	}

	// Feature: NFC
	var identify func() (string, error)
	if wb.hasFeature(warp.FeatureNfc) {
		identify = wb.identify
	}

	// only setup phase switching methods if power manager endpoint is set
	var phases func(int) error
	var getPhases func() (int, error)
	if wb.pmURI != "" {
		if res, err := wb.ensurePmState(); err == nil && res.ExternalControl != warp.ExternalControlDeactivated {
			wb.pmState = res
			phases = wb.phases1p3p
			getPhases = wb.getPhases
		}
	}

	// Phase Auto Switching needs to be disabled for WARP3
	// Necessary if charging 1p only vehicles
	if cc.DisablePhaseAutoSwitch {
		// unfortunately no feature to check for, instead this is set in template
		if err := wb.disablePhaseAutoSwitch(); err != nil {
			return nil, err
		}
		wb.log.TRACE.Println("disabled phase auto switching")
	}

	return decorateWarpWS(wb, currentPower, totalEnergy, currents, voltages, identify, phases, getPhases), nil
}

func NewWarpWS(ctx context.Context, uri, user, password string, meterIndex uint) (*WarpWS, error) {
	log := util.NewLogger("warp-ws")

	client := request.NewHelper(log)
	if user != "" {
		client.Client.Transport = digest.NewTransport(user, password, client.Client.Transport)
	}

	w := &WarpWS{
		Helper: client, log: log,
		uri:                 util.DefaultScheme(uri, "http"),
		meterIndex:          meterIndex,
		meterMap:            map[int]int{},
		metersValueIDsTopic: fmt.Sprintf("meters/%d/value_ids", meterIndex),
		metersValuesTopic:   fmt.Sprintf("meters/%d/values", meterIndex),
	}

	go w.run(ctx)

	return w, nil
}

func (w *WarpWS) run(ctx context.Context) {
	uri, err := parseURI(w.uri, true)
	if err != nil {
		w.log.DEBUG.Println(err)
		if uri == "" {
			return
		}
	}
	w.log.TRACE.Printf("connecting to %s …", uri)

	bo := backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(0))
	for ctx.Err() == nil {
		conn, _, err := websocket.Dial(ctx, uri, nil)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(bo.NextBackOff())
			continue
		}

		bo.Reset()
		if err := w.handleConnection(ctx, conn); err != nil {
			w.log.ERROR.Println(err)
		}
	}
}

func parseURI(uri string, toWS bool) (string, error) {
	u, err := url.Parse(util.DefaultScheme(strings.TrimRight(uri, "/"), "http"))
	if err != nil {
		return "", err
	}
	if u.Scheme == "https" || u.Scheme == "wss" {
		u.Scheme = "http"
		err = fmt.Errorf("https or wss are not supported, using http/ws instead")
	}
	if toWS {
		u.Scheme = "ws"
		u.Path = path.Join(u.Path, "/ws")
	}
	return u.String(), err
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
			var event warpEvent
			if err := dec.Decode(&event); err != nil {
				if errors.Is(err, io.EOF) {
					break //next frame
				}
				return err
			}

			w.log.TRACE.Printf("ws event %s: %s", event.Topic, event.Payload)
			if err := w.handleEvent(event.Topic, event.Payload); err != nil {
				w.log.ERROR.Printf("bad payload for topic %s: %v", event.Topic, err)
			}
		}
	}
}

func (w *WarpWS) handleEvent(topic string, payload json.RawMessage) error {
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
		if !slices.Contains(w.features, warp.FeatureMeterAllValues) || slices.Contains(w.features, warp.FeatureMeters) {
			return nil
		}
		err = json.Unmarshal(payload, &w.meter.TmpValues)
		if len(w.meter.TmpValues) > 5 {
			copy(w.meter.Voltages[:], w.meter.TmpValues[:3])
			copy(w.meter.Currents[:], w.meter.TmpValues[3:6])
		}
	case "meter/values":
		if !slices.Contains(w.features, warp.FeatureMeter) || slices.Contains(w.features, warp.FeatureMeters) {
			return nil
		}
		err = json.Unmarshal(payload, &w.meter)
	case w.metersValueIDsTopic:
		var ids []int
		if err = json.Unmarshal(payload, &ids); err != nil {
			return err
		}
		w.meterMap = make(map[int]int, len(ids))
		for i, id := range ids {
			w.meterMap[id] = i
		}
	case w.metersValuesTopic:
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
	w.mu.RLock()
	if w.features != nil {
		w.mu.RUnlock()
		return slices.Contains(w.features, feature)
	}
	uri := fmt.Sprintf("%s/info/features", w.uri)
	w.mu.RUnlock()

	var f []string
	if err := w.GetJSON(uri, &f); err == nil {
		w.mu.Lock()
		w.features = f
		w.mu.Unlock()
		return slices.Contains(f, feature)
	}

	return false
}

func (w *WarpWS) Enable(enable bool) error {
	var curr int64
	if enable {
		w.mu.RLock()
		curr = w.maxCurrent
		w.mu.RUnlock()
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

// MaxCurrentMillis implements the api.ChargerEx interface
func (w *WarpWS) MaxCurrentMillis(current float64) error {
	curr := int64(current * 1e3)
	err := w.setCurrent(curr)
	if err == nil {
		w.mu.Lock()
		w.maxCurrent = curr
		w.mu.Unlock()
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
	uri := fmt.Sprintf("%s/evse/external_current", w.uri)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(map[string]int64{"current": curr}), request.JSONEncoding)
	_, err := w.Do(req)
	return err
}

func (w *WarpWS) disablePhaseAutoSwitch() error {
	uri := fmt.Sprintf("%s/evse/phase_auto_switch", w.uri)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(map[string]bool{"enabled": false}), request.JSONEncoding)
	_, err := w.Do(req)
	return err
}

// phases1p3p implements the api.PhaseSwitcher interface
func (w *WarpWS) phases1p3p(phases int) error {
	if ec, err := w.ensurePmState(); err != nil || ec.ExternalControl > warp.ExternalControlAvailable {
		return fmt.Errorf("external control not available: %d", ec.ExternalControl)
	}
	w.mu.RLock()
	em := w.pmHelper
	uri := fmt.Sprintf("%s/power_manager/external_control", w.pmURI)
	w.mu.RUnlock()

	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(map[string]int{"phases_wanted": phases}), request.JSONEncoding)

	if em != nil {
		_, err := em.Do(req)
		return err
	}

	_, err := w.Do(req)
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
	em := w.pmHelper
	uri := w.pmURI
	w.mu.RUnlock()
	if em == nil || uri == "" {
		return s, nil
	}

	var ns warp.PmLowLevelState
	if err := em.GetJSON(fmt.Sprintf("%s/power_manager/low_level_state", uri), &ns); err != nil {
		return warp.PmLowLevelState{}, err
	}

	w.mu.Lock()
	w.pmLowLevelState = ns
	w.mu.Unlock()

	return ns, nil
}

func (w *WarpWS) ensurePmState() (warp.PmState, error) {
	w.mu.RLock()
	s := w.pmState
	em := w.pmHelper
	uri := w.pmURI
	w.mu.RUnlock()

	if em == nil || uri == "" || s.ExternalControl != warp.ExternalControlAvailable {
		return s, nil
	}

	var ns warp.PmState
	if err := em.GetJSON(fmt.Sprintf("%s/power_manager/state", uri), &ns); err != nil {
		return warp.PmState{}, err
	}

	w.mu.Lock()
	w.pmState = ns
	w.mu.Unlock()

	return ns, nil
}
