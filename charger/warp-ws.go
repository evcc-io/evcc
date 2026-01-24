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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/warp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/gorilla/websocket"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

type WarpWS struct {
	*request.Helper
	log       *util.Logger
	uri       string
	valuesMap warp.MeterValuesIndices
	skipLegacy bool

	mu sync.RWMutex

	// evse
	status     api.ChargeStatus
	maxCurrent int64

	// meter
	power  float64
	energy float64
	currL  [3]float64
	voltL  [3]float64
	meterIndex uint

	// rfid
	tagId     string
	nfcConfig warp.NfcConfig

	// energy manager
	emStateG    warp.EmState         // analog zu emStateG
	emLowLevel  warp.EmLowLevelState // analog zu emLowLevelG
	is3Phase    bool
	emURI       string
	emHelper    *request.Helper

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

    wb, err := NewWarpWS(cc.URI, cc.User, cc.Password)
    if err != nil {
        return nil, err
    }

    // Feature: Meter
    var currentPower, totalEnergy func() (float64, error)
	if wb.hasFeature(warp.FeatureMeter) {
		currentPower = wb.currentPower
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
    if wb.emStateG.ExternalControl != 1 {
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


func NewWarpWS(uri, user, password string) (*WarpWS, error) {
	log := util.NewLogger("warp-ws")

	client := request.NewHelper(log)
	if user != "" {
		client.Client.Transport = digest.NewTransport(user, password, client.Client.Transport)
	}

	w := &WarpWS{
		Helper: client, log: log,
		uri:     util.DefaultScheme(uri, "http"),
		current: 6000,
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go w.runWS(ctx)

	return w, nil
}

func (w *WarpWS) runWS(ctx context.Context) {
	wsURL := strings.Replace(w.uri, "http://", "ws://", 1) + "/ws"
	for {
		err := w.connectWS(ctx, wsURL)
		if err != nil {
			w.log.ERROR.Printf("ws error: %v", err)
		}
		time.Sleep(3 * time.Second)
	}
}

func (w *WarpWS) connectWS(ctx context.Context, url string) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	w.log.INFO.Printf("connected to WARP websocket")
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		w.handleFrame(data)
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
	if err != nil {
		w.log.ERROR.Printf("ws split error: %v", err)
		return
	}

	for _, obj := range objs {
		w.handleEvent(obj)
	}
}

func (w *WarpWS) handleEvent(data []byte) {
	var evt warpEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		w.log.ERROR.Printf("ws decode: %v", err)
		return
	}
	w.log.TRACE.Printf("Received WARP event with topic: %s and payload: %v", evt.Topic, string(evt.Payload))

	w.mu.Lock()
	defer w.mu.Unlock()

	switch evt.Topic {

	case "charge_tracker/current_charge":
		var c warp.ChargeTrackerCurrentCharge
		if err := json.Unmarshal(evt.Payload, &c); err != nil {
			w.log.ERROR.Printf("charge_tracker decode: %v", err)
			return
		}
		w.tagId = c.AuthorizationInfo.TagId

	case "evse/state":
		var s warp.EvseState
		if err := json.Unmarshal(evt.Payload, &s); err != nil {
			w.log.ERROR.Printf("evse/state decode: %v", err)
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

	case "evse/external_current":
		var c struct {
			Current int64 `json:"current"`
		}
		if err := json.Unmarshal(evt.Payload, &c); err == nil {
			w.maxCurrent = c.Current
		}

	case "meter/values":
		if w.skipLegacy {
			return
		}
		var m warp.MeterValues
		if err := json.Unmarshal(evt.Payload, &m); err != nil {
			w.log.ERROR.Printf("meter/values decode: %v", err)
			return
		}
		w.power = m.Power
		w.energy = m.EnergyAbs

	case "meter/all_values":
		if w.skipLegacy {
			return
		}
		var vals []float64
		if err := json.Unmarshal(evt.Payload, &vals); err != nil {
			w.log.ERROR.Printf("meter/all_values decode: %v", err)
			return
		}
		
		if len(vals) >= 6 {
			w.voltL[0], w.voltL[1], w.voltL[2] = vals[0], vals[1], vals[2]
			w.currL[0], w.currL[1], w.currL[2] = vals[3], vals[4], vals[5]
		}

	case fmt.Sprintf("meters/%d/value_ids", w.meterIndex):
		if !w.skipLegacy {
			w.skipLegacy = true
		}
		var ids []int
		if err := json.Unmarshal(evt.Payload, &ids); err != nil {
			w.log.ERROR.Printf("value_ids decode: %v", err)
		    return
		}
		w.updateMeterValueIds(ids)

	case fmt.Sprintf("meters/%d/values", w.meterIndex):
    	var vals []float64
    	if err := json.Unmarshal(evt.Payload, &vals); err != nil {
        	w.log.ERROR.Printf("values decode: %v", err)
        	return
    	}
    	w.updateMeterValues(vals)

	case "nfc/config":
		var s warp.NfcConfig
		if err := json.Unmarshal(evt.Payload, &s); err != nil {
			w.log.ERROR.Printf("values decode: %v", err)
			return
		}
		w.nfcConfig = s

	case "power_manager/state":
		var s warp.EmState
		if err := json.Unmarshal(evt.Payload, &s); err != nil {
			w.log.ERROR.Printf("em state decode: %v", err)
			return
		}
		w.emStateG = s

	case "power_manager/low_level_state":
		var s warp.EmLowLevelState
		if err := json.Unmarshal(evt.Payload, &s); err != nil {
			w.log.ERROR.Printf("em low_level decode: %v", err)
			return
		}
		w.emLowLevel = s
		w.is3Phase = s.Is3phase
	}
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
	return w.status != api.StatusA, nil
}

func (w *WarpWS) MaxCurrent(current int64) error {
	w.maxCurrent = current
	return w.setCurrent(current)
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

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return fmt.Errorf("disabling phase auto switch failed: %v", err)
	}

	if _, err := w.Do(req); err != nil {
		return fmt.Errorf("disabling phase auto switch failed: %v", err)
	}
	return nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (w *WarpWS) phases1p3p(phases int) error {

	if w.emStateG.ExternalControl > warp.ExternalControlAvailable {
		return fmt.Errorf("external control not available: %s", w.emStateG.ExternalControl.String())
	}

	uri := fmt.Sprintf("%s/power_manager/external_control", w.uri)
	data := map[string]int{"phases_wanted": phases}

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

func (w *WarpWS) currentPower() (float64, error) { return w.CurrentPower() }
func (w *WarpWS) totalEnergy() (float64, error) { return w.TotalEnergy() }
func (w *WarpWS) currents() (float64, float64, float64, error) { return w.Currents() }
func (w *WarpWS) voltages() (float64, float64, float64, error) { return w.Voltages() }
func (w *WarpWS) identify() (string, error) { return w.Identify() }
func (w *WarpWS) updateMeterValueIds(res []int) {
    required := []int{
        warp.ValueIDVoltageL1N,
        warp.ValueIDVoltageL2N,
        warp.ValueIDVoltageL3N,
        warp.ValueIDCurrentImExSumL1,
        warp.ValueIDCurrentImExSumL2,
        warp.ValueIDCurrentImExSumL3,
        warp.ValueIDPowerImExSum,
        warp.ValueIDEnergyAbsImSum,
    }

    // prüfen ob alle IDs vorhanden sind
    missing := []int{}
    for _, req := range required {
        found := false
        for _, id := range res {
            if id == req {
                found = true
                break
            }
        }
        if !found {
            missing = append(missing, req)
        }
    }

    if len(missing) > 0 {
        w.log.ERROR.Printf("missing required meter value IDs: %v", missing)
        return
    }

    // Mapping erzeugen
    var idx warp.MeterValuesIndices
    for i, valueID := range res {
        switch valueID {
        case warp.ValueIDVoltageL1N:
            idx.VoltageL1NIndex = i
        case warp.ValueIDVoltageL2N:
            idx.VoltageL2NIndex = i
        case warp.ValueIDVoltageL3N:
            idx.VoltageL3NIndex = i
        case warp.ValueIDCurrentImExSumL1:
            idx.CurrentImExSumL1Index = i
        case warp.ValueIDCurrentImExSumL2:
            idx.CurrentImExSumL2Index = i
        case warp.ValueIDCurrentImExSumL3:
            idx.CurrentImExSumL3Index = i
        case warp.ValueIDPowerImExSum:
            idx.PowerImExSumIndex = i
        case warp.ValueIDEnergyAbsImSum:
            idx.EnergyAbsImSumIndex = i
        }
    }

    w.valuesMap = idx
    w.log.INFO.Printf("meter value_ids mapped: %+v", idx)
}

func (w *WarpWS) updateMeterValues(res []float64) {
	highestIndex := max(w.valuesMap.CurrentImExSumL1Index, w.valuesMap.VoltageL2NIndex, w.valuesMap.VoltageL3NIndex,
	w.valuesMap.CurrentImExSumL1Index, w.valuesMap.CurrentImExSumL2Index, w.valuesMap.CurrentImExSumL3Index,
	w.valuesMap.PowerImExSumIndex, w.valuesMap.EnergyAbsImSumIndex)

	if len(res) < highestIndex + 1 {
		return
	}

    w.voltL[0] = res[w.valuesMap.VoltageL1NIndex]
	w.voltL[1] = res[w.valuesMap.VoltageL2NIndex]
	w.voltL[2] = res[w.valuesMap.VoltageL3NIndex]
	w.currL[0] = res[w.valuesMap.CurrentImExSumL1Index]
	w.currL[1] = res[w.valuesMap.CurrentImExSumL2Index]
	w.currL[2] = res[w.valuesMap.CurrentImExSumL3Index]
	w.power = res[w.valuesMap.PowerImExSumIndex]
	w.energy = res[w.valuesMap.EnergyAbsImSumIndex]
}

func (w *WarpWS) emState() (warp.EmState, error) {
    w.mu.RLock()
    defer w.mu.RUnlock()
    return w.emStateG, nil
}

