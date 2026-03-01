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
	"github.com/icholy/digest"
)

type WarpWS struct {
	*warp.Connection
	PM *warp.Connection

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
	pmState         warp.PmState
	pmLowLevelState warp.PmLowLevelState
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
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewWarpWS(ctx, cc.URI, cc.User, cc.Password, cc.EnergyManagerURI, cc.EnergyManagerUser, cc.EnergyManagerPassword, cc.EnergyMeterIndex)
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

	// Feature: NFC
	var identify func() (string, error)
	if wb.hasFeature(warp.FeatureNfc) {
		identify = wb.identify
	}

	// Feature: Phase Switching
	if wb.hasFeature(warp.FeaturePhaseSwitch) && wb.PM == nil {
		wb.PM = wb.Connection // Energy Manager not needed, charger will do the phase switching
	}

	// only setup phase switching methods if power manager endpoint is set
	var phases func(int) error
	var getPhases func() (int, error)
	if wb.PM != nil {
		if res, err := wb.ensurePmState(); err == nil && res.ExternalControl != warp.ExternalControlDeactivated {
			wb.pmState = res
			phases = wb.phases1p3p
			getPhases = wb.getPhases
		}
	}

	// Phase Auto Switching needs to be disabled for WARP3 and WARP2 + EM
	// Necessary if charging 1p only vehicles
	typ, err := wb.getWarpType()
	if err != nil {
		return nil, err
	}
	if typ == "warp3" || (typ == "warp2" && wb.PM != nil && wb.PM != wb.Connection) {
		if err := wb.disablePhaseAutoSwitch(); err != nil {
			return nil, err
		}
		wb.log.TRACE.Println("disabled phase auto switching")
	}

	return decorateWarpWS(wb, currentPower, totalEnergy, currents, voltages, identify, phases, getPhases), nil
}

func NewWarpWS(ctx context.Context, uri, user, pass, emURI, emUser, emPass string, meterIndex uint) (*WarpWS, error) {
	log := util.NewLogger("warp-ws")

	c := warp.NewConnection(log, uri, user, pass)
	var pm *warp.Connection
	if emURI != "" {
		pm = warp.NewConnection(log, emURI, emUser, emPass)
	}

	w := &WarpWS{
		Connection: c,
		PM:         pm,
		log:        log,
		meterIndex: meterIndex,
		meterMap:   map[int]int{},
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

		conn, err := dialWebsocket(ctx, digest.Options{
			URI:      wsURI,
			Username: w.Username,
			Password: w.Password,
		})
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

func dialWebsocket(ctx context.Context, options digest.Options) (*websocket.Conn, error) {
	// err will be non nil if auth is needed
	conn, resp, err := websocket.Dial(ctx, options.URI, nil)
	if err == nil {
		return conn, nil
	}

	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		return nil, err
	}

	if options.Username == "" {
		return nil, errors.New("websocket: missing credentials")
	}

	// extract challenge from response
	challenge, err := digest.ParseChallenge(resp.Header.Get("WWW-Authenticate"))
	if err != nil {
		return nil, fmt.Errorf("websocket: %w", err)
	}

	options.Method = "GET"
	options.Count = 1

	cred, err := digest.Digest(challenge, options)
	if err != nil {
		return nil, err
	}

	// Dial with Digest Auth
	dialer := websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": []string{cred.String()},
		},
	}

	conn, _, err = websocket.Dial(ctx, options.URI, &dialer)
	return conn, err
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
	w.mu.RLock()
	if w.features != nil {
		w.mu.RUnlock()
		return slices.Contains(w.features, feature)
	}
	uri := fmt.Sprintf("%s/info/features", w.URI)
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

var _ api.ChargerEx = (*WarpWS)(nil)

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
	if ec, err := w.ensurePmState(); err != nil || ec.ExternalControl > warp.ExternalControlAvailable {
		return fmt.Errorf("external control not available: %d", ec.ExternalControl)
	}
	w.mu.RLock()
	em := w.PM
	w.mu.RUnlock()

	// Should point to warp connection or energy manager connection
	if em == nil {
		return fmt.Errorf("Power Manager endpoint shouldn't be nil")
	}

	uri := fmt.Sprintf("%s/power_manager/external_control", em.URI)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(map[string]int{"phases_wanted": phases}), request.JSONEncoding)
	_, err := em.Do(req)
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
	em := w.PM
	w.mu.RUnlock()
	if em == nil {
		return s, nil
	}

	var ns warp.PmLowLevelState
	if err := em.GetJSON(fmt.Sprintf("%s/power_manager/low_level_state", em.URI), &ns); err != nil {
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
	em := w.PM
	w.mu.RUnlock()

	if em == nil || s.ExternalControl != warp.ExternalControlAvailable {
		return s, nil
	}

	var ns warp.PmState
	if err := em.GetJSON(fmt.Sprintf("%s/power_manager/state", em.URI), &ns); err != nil {
		return warp.PmState{}, err
	}

	w.mu.Lock()
	w.pmState = ns
	w.mu.Unlock()

	return ns, nil
}

func (w *WarpWS) getWarpType() (string, error) {
	var res warp.Name
	uri := fmt.Sprintf("%s/info/name", w.URI)
	err := w.GetJSON(uri, &res)
	return res.WarpType, err
}
