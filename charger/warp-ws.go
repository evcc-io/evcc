package charger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

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
	log           *util.Logger
	uri           string
	inBulkDump    bool
	eventHandlers map[string]handlerFn

	mu sync.RWMutex

	// capabilities
	features []string

	// evse
	evse       warp.Evse
	maxCurrent int64 // input from evcc

	// meter
	meter       warp.MeterVals
	meterIndex  uint
	meterMapper *warp.MeterMapper

	// nfc
	tagId     string
	nfcConfig warp.NfcConfig

	// power manager
	pmState         warp.PmState
	pmLowLevelState warp.PmLowLevelState
	pmURI           string
	pmHelper        *request.Helper

	// config
	current int64
}

type warpEvent struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

type handlerFn func(*WarpWS, json.RawMessage)

var _ api.ChargerEx = (*WarpWS)(nil)

func init() {
	registry.AddCtx("warp-ws", NewWarpWSFromConfig)
}

//go:generate go tool decorate -f decorateWarpWS -b *WarpWS -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

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

	wb, err := NewWarpWS(cc.URI, cc.User, cc.Password, cc.EnergyMeterIndex)
	if err != nil {
		return nil, err
	}

	// Feature: Meter
	var currentPower, totalEnergy func() (float64, error)
	if wb.hasFeature(warp.FeatureMeter) || wb.hasFeature(warp.FeatureMeters) {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
	}

	// Feature: Phases
	var currents, voltages func() (float64, float64, float64, error)
	if wb.hasFeature(warp.FeatureMeters) || wb.hasFeature(warp.FeatureMeterAllValues) {
		currents = wb.currents
		voltages = wb.voltages
	}

	if wb.hasFeature(warp.FeaturePhaseSwitch) && wb.pmURI == "" {
		wb.pmURI = wb.uri
		wb.pmHelper = wb.Helper
	} else if cc.EnergyManagerURI != "" { // fallback to Energy Manager
		wb.pmURI = util.DefaultScheme(strings.TrimRight(cc.EnergyManagerURI, "/"), "http")
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

	// Feature: EM
	var phases func(int) error
	var getPhases func() (int, error)
	if wb.pmState.ExternalControl != 1 {
		phases = wb.phases1p3p
		getPhases = wb.getPhases
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

	return decorateWarpWS(
		wb,
		currentPower,
		totalEnergy,
		currents,
		voltages,
		identify,
		phases,
		getPhases,
	), nil
}

func NewWarpWS(ctx context.Context, uri, user, password string, meterIndex uint) (*WarpWS, error) {
	log := util.NewLogger("warp-ws")

	client := request.NewHelper(log)
	if user != "" {
		client.Client.Transport = digest.NewTransport(user, password, client.Client.Transport)
	}

	w := &WarpWS{
		Helper: client, log: log,
		uri:         util.DefaultScheme(uri, "http"),
		current:     0,
		meterIndex:  meterIndex,
		meterMapper: &warp.MeterMapper{Log: log},
		inBulkDump:  true,
	}

	go w.run(ctx)

	return w, nil
}

func (w *WarpWS) run(ctx context.Context) {
	uri := strings.Replace(w.uri, "http://", "ws://", 1) + "/ws"
	w.log.TRACE.Printf("ws: connecting to %s …", uri)

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // never stop retrying

	operation := func() error {
		conn, _, err := websocket.Dial(ctx, uri, nil)
		if err != nil {
			w.log.ERROR.Printf("ws dial error: %v", err)
			return err
		}

		w.log.DEBUG.Println("ws: connection established")
		conn.SetReadLimit(-1)

		// Read loop
		for {
			typ, data, err := conn.Read(ctx)
			if err != nil {
				w.log.DEBUG.Printf("ws read error: %v", err)
				_ = conn.Close(websocket.StatusInternalError, "read error")
				return err
			}

			if typ == websocket.MessageBinary || typ == websocket.MessageText {
				w.handleFrame(data)
			}
		}
	}

	// Retry forever until ctx is done
	_ = backoff.Retry(operation, backoff.WithContext(bo, ctx))

	w.log.DEBUG.Println("ws: stopping reconnect loop")
}

func splitJSONObjects(data []byte) ([][]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var objs [][]byte
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				return objs, nil
			}
			return nil, err
		}
		objs = append(objs, raw)
	}
}

func (w *WarpWS) handleFrame(frame []byte) {
	trim := bytes.TrimSpace(frame)

	// Initial bulk dump
	if w.inBulkDump {
		objs, err := splitJSONObjects(trim)
		if err != nil {
			w.log.DEBUG.Printf("ws split error: %v", err)
			return
		}

		// if only 1 object detected -> switch to delta mode
		if len(objs) == 1 {
			w.inBulkDump = false
		}

		for _, obj := range objs {
			w.handleEvent(obj)
		}
		return
	}

	// delta mode
	// expecting exactly one item
	if len(trim) > 0 && trim[0] == '{' {
		w.handleEvent(trim)
		return
	}

	// FALLBACK: unexpected multi-frame in delta mode → back to bulk dump mode
	w.log.DEBUG.Println("ws: unexpected multi-object frame in delta mode, re-enabling splitter")
	w.inBulkDump = true
	w.handleFrame(trim)
}

func decode[T any](payload json.RawMessage, log *util.Logger, topic string) (T, bool) {
	var v T
	if err := json.Unmarshal(payload, &v); err != nil {
		log.DEBUG.Printf("%s decode: %v", topic, err)
		return v, false
	}
	return v, true
}

func withWrite(w *WarpWS, fn func(*WarpWS)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	fn(w)
}

func handle[T any](w *WarpWS, payload json.RawMessage, topic string, apply func(*WarpWS, T)) {
	v, ok := decode[T](payload, w.log, topic)
	if !ok {
		return
	}
	withWrite(w, func(w *WarpWS) { apply(w, v) })
}

func handlerBuilder[T any](topic string, apply func(*WarpWS, T)) handlerFn {
	return func(w *WarpWS, p json.RawMessage) {
		handle(w, p, topic, apply)
	}
}

func (w *WarpWS) buildHandlerMap() map[string]handlerFn {
	h := map[string]handlerFn{
		"evse/state":                    handlerBuilder("evse/state", func(w *WarpWS, s warp.EvseState) { w.evse.State = s }),
		"evse/user_enabled":             handlerBuilder("evse/user_enabled", func(w *WarpWS, b struct{ Enabled bool }) { w.evse.UserEnabled = b.Enabled }),
		"evse/user_current":             handlerBuilder("evse/user_current", func(w *WarpWS, s warp.EvseExternalCurrent) { w.evse.UserCurrent = int64(s.Current) }),
		"evse/external_current":         handlerBuilder("evse/external_current", func(w *WarpWS, s warp.EvseExternalCurrent) { w.evse.ExternalCurrent = int64(s.Current) }),
		"charge_tracker/current_charge": handlerBuilder("charge_tracker/current_charge", func(w *WarpWS, s warp.ChargeTrackerCurrentCharge) { w.tagId = s.AuthorizationInfo.TagId }),
		"nfc/config":                    handlerBuilder("nfc/config", func(w *WarpWS, s warp.NfcConfig) { w.nfcConfig = s }),
	}
	if w.hasFeature(warp.FeatureMeters) {
		addMetersHandlers(h, w)
	} else {
		addOldMeterHandlers(h)
	}
	if w.pmURI == "" {
		h["power_manager/state"] = handlerBuilder("power_manager/state", func(w *WarpWS, s warp.PmState) { w.pmState = s })
		h["power_manager/low_level_state"] = handlerBuilder("power_manager/low_level_state", func(w *WarpWS, s warp.PmLowLevelState) { w.pmLowLevelState = s })
	}
	return h
}

func addOldMeterHandlers(h map[string]handlerFn) {
	h["meter/values"] = handlerBuilder("meter/values", func(w *WarpWS, m warp.MeterValues) { w.meter.Power = m.Power; w.meter.Energy = m.EnergyAbs })
	h["meter/all_values"] = handlerBuilder("meter/all_values", func(w *WarpWS, vals []float64) {
		copy(w.meter.Voltages[:], vals[:3])
		copy(w.meter.Currents[:], vals[3:6])
	})
}

func addMetersHandlers(h map[string]handlerFn, w *WarpWS) {
	// NEW meters API
	h[fmt.Sprintf("meters/%d/value_ids", w.meterIndex)] = handlerBuilder("meters/value_ids", func(w *WarpWS, ids []int) { w.meterMapper.UpdateValueIDs(ids) })
	h[fmt.Sprintf("meters/%d/values", w.meterIndex)] = handlerBuilder("meters/values", func(w *WarpWS, vals []float64) {
		w.meterMapper.UpdateValues(vals, &w.meter.Power, &w.meter.Energy, &w.meter.Voltages, &w.meter.Currents)
	})
}

func (w *WarpWS) handleEvent(data []byte) {
	if w.eventHandlers == nil {
		w.eventHandlers = w.buildHandlerMap()
	}
	evt, ok := decode[warpEvent](data, w.log, "warp event")
	if !ok {
		return
	}
	if h, ok := w.eventHandlers[evt.Topic]; ok {
		h(w, evt.Payload)
	}
}

func (w *WarpWS) hasFeature(feature string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.features != nil {
		return slices.Contains(w.features, feature)
	}

	uri := fmt.Sprintf("%s/info/features", w.uri)
	var features []string
	if err := w.GetJSON(uri, &features); err == nil {
		w.features = features
		return slices.Contains(w.features, feature)
	}

	return false
}

func getField[T any](w *WarpWS, f func(*WarpWS) T) T {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return f(w)
}

func (w *WarpWS) Enable(enable bool) error {
	curr := int64(0)
	if enable {
		curr = getField(w, func(w *WarpWS) int64 { return w.maxCurrent })
	}
	return w.setCurrent(curr)
}

func (w *WarpWS) Enabled() (bool, error) {
	return getField(w, func(w *WarpWS) int64 { return w.evse.ExternalCurrent }) > 0, nil
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
		withWrite(w, func(ww *WarpWS) { w.maxCurrent = curr })
	} else {
		w.log.DEBUG.Printf("evse: set current failed: %v", err)
	}
	return err
}

func (w *WarpWS) statusFromEvseStatus() api.ChargeStatus {
	// TODO only A-C are supported
	evseStatus := []api.ChargeStatus{api.StatusA, api.StatusB, api.StatusC, api.StatusNone, api.StatusE}
	status := getField(w, func(w *WarpWS) warp.EvseState { return w.evse.State })
	if status.Iec61851State < len(evseStatus) {
		return evseStatus[status.Iec61851State]
	}
	// TODO no status MUST be an error
	return api.StatusNone
}

func (w *WarpWS) Status() (api.ChargeStatus, error) {
	return w.statusFromEvseStatus(), nil
}

func (w *WarpWS) StatusReason() (api.Reason, error) {
	status := w.statusFromEvseStatus()
	w.mu.RLock()
	defer w.mu.RUnlock()
	if status == api.StatusB && w.evse.UserEnabled && w.evse.UserCurrent == 0 {
		return api.ReasonWaitingForAuthorization, nil
	}
	return api.ReasonUnknown, nil
}

func (w *WarpWS) currentPower() (float64, error) {
	return getField(w, func(w *WarpWS) float64 { return w.meter.Power }), nil
}

func (w *WarpWS) totalEnergy() (float64, error) {
	return getField(w, func(w *WarpWS) float64 { return w.meter.Energy }), nil
}

func (w *WarpWS) currents() (float64, float64, float64, error) {
	vals := getField(w, func(w *WarpWS) [3]float64 { return w.meter.Currents })
	return vals[0], vals[1], vals[2], nil
}

func (w *WarpWS) voltages() (float64, float64, float64, error) {
	vals := getField(w, func(w *WarpWS) [3]float64 { return w.meter.Voltages })
	return vals[0], vals[1], vals[2], nil
}

func (w *WarpWS) identify() (string, error) {
	return getField(w, func(w *WarpWS) string { return w.tagId }), nil
}

func (w *WarpWS) post(path string, body any) error {
	uri := fmt.Sprintf("%s/%s", w.uri, path)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(body), request.JSONEncoding)

	if w.pmHelper != nil && strings.HasPrefix(path, "power_manager/") {
		_, err := w.pmHelper.Do(req)
		return err
	}

	_, err := w.Do(req)
	return err
}

func (w *WarpWS) setCurrent(curr int64) error {
	return w.post("evse/external_current", map[string]int64{"current": curr})
}

func (w *WarpWS) disablePhaseAutoSwitch() error {
	return w.post("evse/phase_auto_switch", map[string]bool{"enabled": false})
}

// phases1p3p implements the api.PhaseSwitcher interface
func (w *WarpWS) phases1p3p(phases int) error {
	w.ensurePmState()
	ec := getField(w, func(w *WarpWS) warp.ExternalControl { return w.pmState.ExternalControl })
	if ec > warp.ExternalControlAvailable {
		return fmt.Errorf("external control not available: %s", ec.String())
	}

	return w.post("power_manager/external_control", map[string]int{"phases_wanted": phases})
}

// getPhases implements the api.PhaseGetter interface
func (w *WarpWS) getPhases() (int, error) {
	w.ensurePmLowLevelState()
	if getField(w, func(w *WarpWS) bool { return w.pmLowLevelState.Is3phase }) {
		return 3, nil
	}
	return 1, nil
}

func ensure[T any](cond bool, uri string, target *T, w *WarpWS) {
	if w.pmHelper == nil || cond {
		return
	}
	var tmp T
	if err := w.pmHelper.GetJSON(uri, &tmp); err == nil {
		withWrite(w, func(w *WarpWS) { *target = tmp })
	}
}

func (w *WarpWS) ensurePmState() {
	ensure(w.pmState.ExternalControl != 0, fmt.Sprintf("%s/power_manager/state", w.pmURI), &w.pmState, w)
}

func (w *WarpWS) ensurePmLowLevelState() {
	ensure(w.pmLowLevelState.Is3phase, fmt.Sprintf("%s/power_manager/low_level_state", w.pmURI), &w.pmLowLevelState, w)
}
