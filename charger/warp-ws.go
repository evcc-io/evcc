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
	log        *util.Logger
	uri        string
	inBulkDump bool

	mu sync.RWMutex

	// capabilities
	hasMeter          bool
	hasMeterPhases    bool
	hasNfc            bool
	hasPhaseSwitching bool

	// evse
	status          api.ChargeStatus
	maxCurrent      int64
	externalCurrent int64
	userEnabled     bool
	userCurrent     int64

	// meter
	power      float64
	energy     float64
	currL      [3]float64
	voltL      [3]float64
	meterIndex uint
	meter      *warp.MeterMapper

	// rfid
	tagId     string
	nfcConfig warp.NfcConfig

	// energy manager
	emState  warp.EmState
	is3Phase bool
	emURI    string
	emHelper *request.Helper

	// config
	current int64
	cancel  context.CancelFunc

	// decorator hooks
	fnCurrentPower func() (float64, error)
	fnTotalEnergy  func() (float64, error)
	fnCurrents     func() (float64, float64, float64, error)
	fnVoltages     func() (float64, float64, float64, error)
	fnIdentify     func() (string, error)
	fnPhases       func(int) error
	fnGetPhases    func() (int, error)
}

var _ api.ChargerEx = (*WarpWS)(nil)

type warpEvent struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

func init() {
	registry.Add("warp-ws", NewWarpWSFromConfig)
}

//go:generate go tool decorate -f decorateWarpWS -b *WarpWS -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

func NewWarpWSFromConfig(other map[string]any) (api.Charger, error) {
	var cc = struct {
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
	if wb.hasFeature(warp.FeatureMeter) {
		wb.fnCurrentPower = func() (float64, error) { return wb.power, nil }
		wb.fnTotalEnergy = func() (float64, error) { return wb.energy, nil }
		wb.hasMeter = true
	}

	// Feature: Phases
	if wb.hasFeature(warp.FeatureMeterPhases) {
		wb.fnCurrents = func() (float64, float64, float64, error) { return wb.currL[0], wb.currL[1], wb.currL[2], nil }
		wb.fnVoltages = func() (float64, float64, float64, error) { return wb.voltL[0], wb.voltL[1], wb.voltL[2], nil }
		wb.hasMeterPhases = true
	}

	if wb.hasFeature(warp.FeaturePhaseSwitch) {
		wb.fnCurrents = func() (float64, float64, float64, error) { return wb.currL[0], wb.currL[1], wb.currL[2], nil }
		wb.fnVoltages = func() (float64, float64, float64, error) { return wb.voltL[0], wb.voltL[1], wb.voltL[2], nil }
	} else if cc.EnergyManagerURI != "" { // fallback to Energy Manager
		wb.emURI = util.DefaultScheme(strings.TrimRight(cc.EnergyManagerURI, "/"), "http")
		wb.emHelper = request.NewHelper(wb.log)
		if cc.EnergyManagerUser != "" {
			wb.emHelper.Client.Transport = digest.NewTransport(cc.EnergyManagerUser, cc.EnergyManagerPassword, wb.emHelper.Client.Transport)
		}
	}

	// Feature: NFC
	if wb.hasFeature(warp.FeatureNfc) {
		wb.fnIdentify = func() (string, error) { return wb.tagId, nil }
		wb.hasNfc = true
	}

	// Feature: EM
	if wb.emState.ExternalControl != 1 {
		wb.fnPhases = wb.phases1p3p
		wb.fnGetPhases = wb.getPhases
		wb.hasPhaseSwitching = true
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
		wb.fnCurrentPower,
		wb.fnTotalEnergy,
		wb.fnCurrents,
		wb.fnVoltages,
		wb.fnIdentify,
		wb.fnPhases,
		wb.fnGetPhases,
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
		current:    0,
		meterIndex: meterIndex,
		meter:      &warp.MeterMapper{Log: log},
		inBulkDump: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go w.run(ctx)

	return w, nil
}

func (w *WarpWS) run(ctx context.Context) {
	wsURL := strings.Replace(w.uri, "http://", "ws://", 1) + "/ws"
	w.log.TRACE.Printf("ws: connecting to %s …", wsURL)

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // never stop retrying

	operation := func() error {
		conn, _, err := websocket.Dial(ctx, wsURL, nil)
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
				break
			}
			return nil, err
		}
		objs = append(objs, raw)
	}

	return objs, nil
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

		// if only 1 object detect -> switch to delta mode
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

func (w *WarpWS) handlers() map[string]func(*WarpWS, json.RawMessage) {
	return map[string]func(*WarpWS, json.RawMessage){
		"evse/state": func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "evse/state", func(w *WarpWS, s warp.EvseState) {
				w.status = map[int]api.ChargeStatus{
					0: api.StatusA,
					1: api.StatusB,
					2: api.StatusC,
				}[s.Iec61851State]
			})
		},

		"evse/user_enabled": func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "evse/user_enabled", func(w *WarpWS, b struct{ Enabled bool }) {
				w.userEnabled = b.Enabled
			})
		},

		"evse/user_current": func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "evse/user_current", func(w *WarpWS, s warp.EvseExternalCurrent) {
				w.userCurrent = int64(s.Current)
			})
		},

		"evse/external_current": func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "evse/external_current", func(w *WarpWS, s warp.EvseExternalCurrent) {
				w.externalCurrent = int64(s.Current)
			})
		},

		"charge_tracker/current_charge": func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "charge_tracker/current_charge", func(w *WarpWS, s warp.ChargeTrackerCurrentCharge) {
				w.tagId = s.AuthorizationInfo.TagId
				w.hasNfc = true
			})
		},

		"nfc/config": func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "nfc/config", func(w *WarpWS, s warp.NfcConfig) {
				w.nfcConfig = s
			})
		},

		"power_manager/state": func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "power_manager/state", func(w *WarpWS, s warp.EmState) {
				w.emState = s
			})
		},

		"power_manager/low_level_state": func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "power_manager/low_level_state", func(w *WarpWS, s warp.EmLowLevelState) {
				w.is3Phase = s.Is3phase
			})
		},

		// Legacy meter
		"meter/values": func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "meter/values", func(w *WarpWS, m warp.MeterValues) {
				w.power = m.Power
				w.energy = m.EnergyAbs
				w.hasMeter = true
			})
		},

		"meter/all_values": func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "meter/all_values", func(w *WarpWS, vals []float64) {
				copy(w.voltL[:], vals[:3])
				copy(w.currL[:], vals[3:6])
				w.hasMeterPhases = true
			})
		},

		// NEW meter API — now integrated!
		fmt.Sprintf("meters/%d/value_ids", w.meterIndex): func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "meters/value_ids", func(w *WarpWS, ids []int) {
				w.meter.UpdateValueIDs(ids)
			})
		},

		fmt.Sprintf("meters/%d/values", w.meterIndex): func(w *WarpWS, p json.RawMessage) {
			handle(w, p, "meters/values", func(w *WarpWS, vals []float64) {
				w.meter.UpdateValues(vals, &w.power, &w.energy, &w.voltL, &w.currL)
				w.hasMeter = true
				w.hasMeterPhases = true
			})
		},
	}
}

func (w *WarpWS) handleEvent(data []byte) {
	evt, ok := decode[warpEvent](data, w.log, "warp event")
	if !ok {
		return
	}

	if h, ok := w.handlers()[evt.Topic]; ok {
		h(w, evt.Payload)
	}
}

func (w *WarpWS) hasFeature(f string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	switch f {
	case warp.FeatureMeter:
		return w.hasMeter
	case warp.FeatureMeterPhases:
		return w.hasMeterPhases
	case warp.FeatureNfc:
		return w.hasNfc
	case warp.FeaturePhaseSwitch:
		return w.hasPhaseSwitching
	default:
		return false
	}
}

func getField[T any](w *WarpWS, f func(*WarpWS) T) T {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return f(w)
}

func (w *WarpWS) Status() (api.ChargeStatus, error) {
	return getField(w, func(w *WarpWS) api.ChargeStatus { return w.status }), nil
}

func (w *WarpWS) StatusReason() (api.Reason, error) {
	if w.needsAuthorization() {
		return api.ReasonWaitingForAuthorization, nil
	}
	return api.ReasonUnknown, nil
}

func (w *WarpWS) CurrentPower() (float64, error) {
	return getField(w, func(w *WarpWS) float64 { return w.power }), nil
}

func (w *WarpWS) TotalEnergy() (float64, error) {
	return getField(w, func(w *WarpWS) float64 { return w.energy }), nil
}

func (w *WarpWS) Currents() (float64, float64, float64, error) {
	vals := getField(w, func(w *WarpWS) [3]float64 { return w.currL })
	return vals[0], vals[1], vals[2], nil
}

func (w *WarpWS) Voltages() (float64, float64, float64, error) {
	vals := getField(w, func(w *WarpWS) [3]float64 { return w.voltL })
	return vals[0], vals[1], vals[2], nil
}

func (w *WarpWS) Identify() (string, error) {
	return getField(w, func(w *WarpWS) string { return w.tagId }), nil
}

func (w *WarpWS) Enable(enable bool) error {
	curr := int64(0)
	if enable {
		curr = getField(w, func(w *WarpWS) int64 { return w.maxCurrent })
	}
	return w.setCurrent(curr)
}

func (w *WarpWS) Enabled() (bool, error) {
	return getField(w, func(w *WarpWS) int64 { return w.externalCurrent }) > 0, nil
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
	externalControl := getField(w, func(ww *WarpWS) warp.ExternalControl { return w.emState.ExternalControl })
	if externalControl > warp.ExternalControlAvailable {
		w.log.DEBUG.Printf("em: external control unavailable (%s)", externalControl.String())
		return fmt.Errorf("external control not available: %s", externalControl.String())
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
	if getField(w, func(w *WarpWS) bool { return w.is3Phase }) {
		return 3, nil
	}
	return 1, nil
}

func (w *WarpWS) needsAuthorization() bool {
	return getField(w, func(w *WarpWS) api.ChargeStatus { return w.status }) == api.StatusB && getField(w, func(w *WarpWS) bool {
		return w.userEnabled
	}) && getField(w, func(w *WarpWS) int64 { return w.userCurrent }) == 0
}
