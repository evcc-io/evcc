package charger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/warp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

type WarpWS struct {
	*request.Helper
	log *util.Logger
	uri string

	mu sync.RWMutex

	// evse
	status     api.ChargeStatus
	maxCurrent int64

	// meter
	power      float64
	energy     float64
	currL      [3]float64
	voltL      [3]float64
	meterIndex uint
	meter      warp.MeterMapper

	// rfid
	tagId     string
	nfcConfig warp.NfcConfig

	// energy manager
	emState    warp.EmState
	emLowLevel warp.EmLowLevelState
	is3Phase   bool
	emURI      string
	emHelper   *request.Helper

	// config
	current int64
	cancel  context.CancelFunc
}

type warpEvent struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

func init() {
	registry.Add("warp-ws", NewWarpWSFromConfig)
}

//go:generate go tool decorate -f decorateWarpWS -b *WarpWS -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

func NewWarpWSFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI                    string
		User                   string
		Password               string
		EnergyManagerURI       string
		EnergyManagerUser      string
		EnergyManagerPassword  string
		DisablePhaseAutoSwitch bool
		EnergyMeterIndex       uint
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewWarpWS(cc.URI, cc.User, cc.Password, cc.EnergyMeterIndex)
	if err != nil {
		return nil, err
	}

	// Feature: Meter
	var currentPower, totalEnergy func() (float64, error)
	if wb.hasFeature(warp.FeatureMeter) {
		currentPower = wb.CurrentPower
		totalEnergy = wb.totalEnergy
	}

	// Feature: Phasen
	var currents, voltages func() (float64, float64, float64, error)

	if wb.hasFeature(warp.FeatureMeterPhases) {
		currents = wb.currents
		voltages = wb.voltages
	}

	if wb.hasFeature(warp.FeaturePhaseSwitch) {
		currents = wb.currents
		voltages = wb.voltages
	} else if cc.EnergyManagerURI != "" { // fallback to Energy Manager
		wb.emURI = util.DefaultScheme(strings.TrimRight(cc.EnergyManagerURI, "/"), "http")
		wb.emHelper = request.NewHelper(wb.log)
		if cc.EnergyManagerUser != "" {
			wb.emHelper.Client.Transport = digest.NewTransport(cc.EnergyManagerUser, cc.EnergyManagerPassword, wb.emHelper.Client.Transport)
		}
	}

	// Feature: NFC
	var identity func() (string, error)
	if wb.hasFeature(warp.FeatureNfc) {
		identity = wb.identify
	}

	// Feature: EM
	var phases func(int) error
	var getPhases func() (int, error)
	if wb.emState.ExternalControl != 1 {
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
		identity,
		phases,
		getPhases,
	), nil
}

func NewWarpWS(uri, user, password string, meterIndex uint) (*WarpWS, error) {
	log := util.NewLogger("warp-ws")

	client := request.NewHelper(log)
	if user != "" {
		client.Client.Transport = digest.NewTransport(user, password, client.Client.Transport)
	}

	w := &WarpWS{
		Helper: client, log: log,
		uri:        util.DefaultScheme(uri, "http"),
		current:    6000,
		meterIndex: meterIndex,
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go w.run(ctx)

	return w, nil
}

func (w *WarpWS) run(ctx context.Context) {
	wsURL := strings.Replace(w.uri, "http://", "ws://", 1) + "/ws"
	w.log.TRACE.Printf("ws: connecting to %s …", wsURL)

	for {
		// Connect
		conn, _, err := websocket.Dial(ctx, wsURL, nil)
		conn.SetReadLimit(-1)
		if err != nil {
			w.log.ERROR.Printf("ws dial error: %v", err)
		} else {
			w.log.DEBUG.Println("ws: connection established")
			// Read-Loop
			for {
				typ, data, err := conn.Read(ctx)
				if err != nil {
					w.log.DEBUG.Printf("ws read error: %v", err)
					_ = conn.Close(websocket.StatusInternalError, "read error")
					break
				}
				if typ == websocket.MessageBinary || typ == websocket.MessageText {
					w.handleFrame(data)
				}
			}
		}

		w.log.TRACE.Println("ws: reconnecting in 3s …")
		// Reconnect handling
		select {
		case <-ctx.Done():
			w.log.DEBUG.Println("ws: stopping reconnect loop")
			return
		case <-time.After(3 * time.Second):
		}
	}
}

func splitJSONObjects(data []byte) ([][]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var objs [][]byte

	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		objs = append(objs, raw)
	}

	return objs, nil
}

func (w *WarpWS) handleFrame(frame []byte) {
	objs, err := splitJSONObjects(frame)
	w.log.TRACE.Printf("ws: frame size=%d bytes, objects=%d", len(frame), len(objs))
	if err != nil {
		w.log.DEBUG.Printf("ws split error: %v", err)
		return
	}

	for _, obj := range objs {
		w.handleEvent(obj)
	}
}

func (w *WarpWS) handleEvent(data []byte) {
	var evt warpEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		w.log.DEBUG.Printf("ws decode: %v", err)
		return
	}

	w.log.TRACE.Printf("Received WARP event with topic: %s", evt.Topic)

	w.mu.Lock()
	defer w.mu.Unlock()

	switch evt.Topic {
	case "charge_tracker/current_charge":
		w.handleChargeTracker(evt.Payload)

	case "evse/state":
		w.handleEvseState(evt.Payload)

	case "evse/external_current":
		w.handleExternalCurrent(evt.Payload)

	case "meter/values":
		w.handleMeterValues(evt.Payload)

	case "meter/all_values":
		w.handleMeterAllValues(evt.Payload)

	case fmt.Sprintf("meters/%d/value_ids", w.meterIndex):
		w.handleValueIDs(evt.Payload)

	case fmt.Sprintf("meters/%d/values", w.meterIndex):
		w.handleMetersValues(evt.Payload)

	case "nfc/config":
		w.handleNfcConfig(evt.Payload)

	case "power_manager/state":
		w.handleEmState(evt.Payload)

	case "power_manager/low_level_state":
		w.handleEmLowLevel(evt.Payload)
	}
}

func (w *WarpWS) decode(payload json.RawMessage, v any, msg string) bool {
	if err := json.Unmarshal(payload, v); err != nil {
		w.log.DEBUG.Printf("%s decode: %v", msg, err)
		return false
	}
	return true
}

func (w *WarpWS) handleChargeTracker(payload json.RawMessage) {
	var c warp.ChargeTrackerCurrentCharge
	if !w.decode(payload, &c, "charge_tracker") {
		return
	}
	w.log.TRACE.Printf("nfc: tag detected: %s", c.AuthorizationInfo.TagId)
	w.tagId = c.AuthorizationInfo.TagId
}

func (w *WarpWS) handleEvseState(payload json.RawMessage) {
	var s warp.EvseState
	if !w.decode(payload, &s, "evse/state") {
		return
	}

	switch s.Iec61851State {
	case 0:
		w.status = api.StatusA
	case 1:
		w.status = api.StatusB
	case 2:
		w.status = api.StatusC
	}
}

func (w *WarpWS) handleExternalCurrent(payload json.RawMessage) {
	var c warp.EvseExternalCurrent
	if w.decode(payload, &c, "evse/external_current") {
		w.log.TRACE.Printf("em: state updated (current=%d)", int64(c.Current))
		w.maxCurrent = int64(c.Current)
	}
}

func (w *WarpWS) handleMeterValues(payload json.RawMessage) {
	var m warp.MeterValues
	if !w.decode(payload, &m, "meter/values") {
		return
	}
	w.power = m.Power
	w.energy = m.EnergyAbs
}

func (w *WarpWS) handleMeterAllValues(payload json.RawMessage) {
	var vals []float64
	if !w.decode(payload, &vals, "meter/all_values") {
		return
	}
	w.meter.HandleLegacyValues(vals, &w.power, &w.energy, &w.voltL, &w.currL)
}

func (w *WarpWS) handleValueIDs(payload json.RawMessage) {
	var ids []int
	if !w.decode(payload, &ids, "value_ids") {
		return
	}
	w.meter.UpdateValueIDs(ids)
}

func (w *WarpWS) handleMetersValues(payload json.RawMessage) {
	var vals []float64
	if !w.decode(payload, &vals, "values") {
		return
	}
	w.meter.UpdateValues(vals, &w.power, &w.energy, &w.voltL, &w.currL)
}

func (w *WarpWS) handleNfcConfig(payload json.RawMessage) {
	var s warp.NfcConfig
	if w.decode(payload, &s, "nfc/config") {
		w.log.DEBUG.Printf("nfc: config updated (config=%v)", s)
		w.nfcConfig = s
	}
}

func (w *WarpWS) handleEmState(payload json.RawMessage) {
	var s warp.EmState
	if w.decode(payload, &s, "power_manager/state") {
		w.emState = s
	}
}

func (w *WarpWS) handleEmLowLevel(payload json.RawMessage) {
	var s warp.EmLowLevelState
	if !w.decode(payload, &s, "power_manager/low_level_state") {
		return
	}
	w.log.TRACE.Printf("em: low-level updated (s=%v) (is3phase=%v)", s, s.Is3phase)
	w.emLowLevel = s
	w.is3Phase = s.Is3phase
}

func (w *WarpWS) hasFeature(f string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	switch f {
	case warp.FeatureMeter:
		return w.power != 0 || w.energy != 0
	case warp.FeatureMeterPhases:
		return w.currL[0] != 0 || w.voltL[0] != 0
	case warp.FeatureNfc:
		return w.tagId != ""
	default:
		w.log.TRACE.Printf("feature missing: %s", f)
		return false
	}
}

func (w *WarpWS) Status() (api.ChargeStatus, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status, nil
}

func (w *WarpWS) CurrentPower() (float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.power, nil
}

func (w *WarpWS) TotalEnergy() (float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.energy, nil
}

func (w *WarpWS) Currents() (float64, float64, float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.currL[0], w.currL[1], w.currL[2], nil
}

func (w *WarpWS) Voltages() (float64, float64, float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.voltL[0], w.voltL[1], w.voltL[2], nil
}

func (w *WarpWS) Identify() (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.tagId, nil
}

func (w *WarpWS) Enable(enable bool) error {
	var curr int64
	if enable {
		curr = w.current
	}
	return w.setCurrent(curr)
}

func (w *WarpWS) Enabled() (bool, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status == api.StatusC, nil
}

// MaxCurrent implements the api.Charger interface
func (w *WarpWS) MaxCurrent(current int64) error {
	return w.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Warp2)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (w *WarpWS) MaxCurrentMillis(current float64) error {
	curr := int64(current * 1e3)
	w.log.TRACE.Printf("evse: setting current=%dmA", curr)
	err := w.setCurrent(curr)
	if err == nil {
		w.log.TRACE.Printf("evse: current set acknowledged (requested=%dmA)", curr)
		w.maxCurrent = curr
	} else {
		w.log.DEBUG.Printf("evse: set current failed: %v", err)
	}
	return err
}

func (w *WarpWS) setCurrent(curr int64) error {
	uri := fmt.Sprintf("%s/evse/external_current", w.uri)
	data := map[string]int64{"current": curr}

	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	_, err := w.Do(req)
	return err
}

func (w *WarpWS) disablePhaseAutoSwitch() error {
	uri := fmt.Sprintf("%s/evse/phase_auto_switch", w.uri)
	data := map[string]bool{"enabled": false}

	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	_, err := w.Do(req)
	return err
}

// phases1p3p implements the api.PhaseSwitcher interface
func (w *WarpWS) phases1p3p(phases int) error {
	if w.emState.ExternalControl > warp.ExternalControlAvailable {
		w.log.DEBUG.Printf("em: external control unavailable (%s)", w.emState.ExternalControl.String())
		return fmt.Errorf("external control not available: %s", w.emState.ExternalControl.String())
	}

	uri := fmt.Sprintf("%s/power_manager/external_control", w.uri)
	data := map[string]int{"phases_wanted": phases}
	w.log.TRACE.Printf("em: switching phases to %dp", phases)

	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	_, err := w.Do(req)
	return err
}

// getPhases implements the api.PhaseGetter interface
func (w *WarpWS) getPhases() (int, error) {
	if w.emLowLevel.Is3phase {
		return 3, nil
	}

	return 1, nil
}

func (w *WarpWS) totalEnergy() (float64, error)                { return w.TotalEnergy() }
func (w *WarpWS) currents() (float64, float64, float64, error) { return w.Currents() }
func (w *WarpWS) voltages() (float64, float64, float64, error) { return w.Voltages() }
func (w *WarpWS) identify() (string, error)                    { return w.Identify() }
