package charger

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	pm *warp.Connection // separate Energy Manager

	implement.Caps

	// config
	log        *util.Logger
	meterIndex uint

	maxCurrent int64 // input from evcc

	mu sync.RWMutex

	// Any data below is protected by mu
	// and should only be trusted if all connErrs are nil.
	// use readWSData (handles both the mutex and the connection errors) to read any data below

	connErrs []error

	// evse
	evse warp.Evse

	// meter
	// warp.MvidValues() is the list of meter value IDs that are of interest for EVCC.
	// indices maps those IDs to the index in the meters/X/values API.
	indices map[warp.Mvid]int
	values  map[warp.Mvid]float64

	// nfc
	chargeTracker warp.ChargeTrackerCurrentCharge

	// ev (WARP4, ISO 15118)
	evState warp.EvState

	// power manager
	pmState         warp.PmState
	pmLowLevelState warp.PmLowLevelState
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

	return w, nil
}

func NewWarpWS(ctx context.Context, uri, user, pass, emURI, emUser, emPass string, meterIndex uint) (*WarpWS, error) {
	log := util.NewLogger("warp-ws")

	// We never use ErrMustRetry anywhere else.
	// This is the marker value for "we've not yet attempted to connect"
	connErrs := []error{api.ErrMustRetry, nil}
	if emURI != "" {
		connErrs[1] = api.ErrMustRetry
	}

	w := &WarpWS{
		Connection: warp.NewConnection(log, uri, user, pass),
		Caps:       implement.New(),
		log:        log,
		meterIndex: meterIndex,
		indices:    make(map[warp.Mvid]int, len(warp.MvidValues())),
		values:     make(map[warp.Mvid]float64, len(warp.MvidValues())),
		connErrs:   connErrs,
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

func (w *WarpWS) setConnectionError(err error, role wsRole) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.connErrs[role] = err
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
			if strings.Contains(err.Error(), "expected handshake response status code 101 but got 401") {
				err = api.ErrMissingCredentials
			} else if errors.Is(err, context.DeadlineExceeded) {
				err = api.ErrTimeout
			}

			w.log.ERROR.Println(err)
			w.setConnectionError(err, role)

			select {
			case <-ctx.Done():
				return
			case <-time.After(bo.NextBackOff()):
			}

			continue
		}

		bo.Reset()

		if err := w.handleConnection(ctx, role, conn); err != nil {
			w.setConnectionError(err, role)
			// TODO: remove this print?
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

		// Split message into lines first:
		// json.Decode eats whitespace, so we can't detect the end of the first dump.
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if len(scanner.Bytes()) == 0 {
				// An empty line signifies the end of the first dump of JSON messages.
				// This dump contains every API exactly once,
				// so now is the time to clear connection errors to indicate
				// that the cached data is valid.
				// Following websocket frames will not contain an empty line.
				w.setConnectionError(nil, role)

				// We could break here, because an empty line will always be the last line in a websocket frame,
				// but continue is sufficient.
				continue
			}

			var event struct {
				Topic   string          `json:"topic"`
				Payload json.RawMessage `json:"payload"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				return err
			}

			if w.pm != w.Connection {
				// If there is a separate connection to an WEM,
				// ignore all messages from the WEM except info/features and the power manager.
				// info/features is accepted from both connections:
				// The WEM declares the phase_switch feture if it is configured correctly.
				// All other features that we check are charger-only
				if role == wsRolePM && event.Topic != "info/features" && !isPmTopic(event.Topic) {
					continue
				}
				// If there is a separate connection to an WEM,
				// ignore messages from the charger's power manager
				if role == wsRoleMain && isPmTopic(event.Topic) {
					continue
				}
			}

			w.log.TRACE.Printf("websocket: event %s: %s", event.Topic, event.Payload)
			if err := w.handleEvent(event.Topic, event.Payload); err != nil {
				return err
			}
		}

		if err := scanner.Err(); err != nil {
			return err
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
		var features []string
		if err = json.Unmarshal(payload, &features); err != nil {
			return err
		}

		var hasFeature = func(feature string) bool {
			return slices.Contains(features, feature)
		}

		// Feature: ISO 15118 (WARP4): vehicle soc and mac exposed via ev/state
		hasIso15118 := hasFeature(warp.FeatureIso15118)
		if hasIso15118 {
			implement.Has(w, implement.Battery(w.soc))
			implement.Has(w, implement.BatteryCapacity(w.capacity))
		}

		// Feature: NFC
		if hasFeature(warp.FeatureNfc) || hasIso15118 {
			implement.Has(w, implement.Identifier(w.identify))
		}

		if hasFeature(warp.FeaturePhaseSwitch) {
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

func (w *WarpWS) Enable(enable bool) error {
	var curr int64
	if enable {
		curr = w.maxCurrent
	}
	return w.setCurrent(curr)
}

func (w *WarpWS) readWSData[R any](fn func() (R, error)) (R, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// If there is an ErrMustRetry, wait until there is not.
	// TODO: Don't sleep here, use a condition variable or similar?
	// Add another second to the deadline to allow some time for the first API dump to arrive.
	deadline := time.Now().Add(max(w.Timeout, w.pm.Timeout)).Add(1 * time.Second)
	for slices.ContainsFunc(w.connErrs, func(e error) bool { return errors.Is(e, api.ErrMustRetry) }) {
		if time.Now().After(deadline) {
			// Return the zero-value of type R if there is a connection error.
			var result R
			return result, api.ErrTimeout
		}

		w.mu.RUnlock()
		time.Sleep(100 * time.Millisecond)
		w.mu.RLock()
	}

	// Return any actual error.
	if idx := slices.IndexFunc(w.connErrs, func(e error) bool {
		return e != nil && !errors.Is(e, api.ErrMustRetry)
	}); idx != -1 {
		// Return the zero-value of type R if there is a connection error.
		var result R
		return result, w.connErrs[idx]
	}

	return fn()
}

func (w *WarpWS) Enabled() (bool, error) {
	return w.readWSData(func() (bool, error) {
		return w.evse.ExternalCurrent.Current > 0, nil
	})
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
	return w.readWSData(func() (api.ChargeStatus, error) {
		return w.statusFromEvseStatus(w.evse.State.Iec61851State)
	})
}

func (w *WarpWS) StatusReason() (api.Reason, error) {
	return w.readWSData(func() (api.Reason, error) {
		if status, err := w.statusFromEvseStatus(w.evse.State.Iec61851State); err != nil {
			return api.ReasonUnknown, err
		} else if status == api.StatusB && w.evse.UserEnabled.Enabled && w.evse.UserCurrent.Current == 0 {
			return api.ReasonWaitingForAuthorization, nil
		}
		return api.ReasonUnknown, nil
	})
}

func (w *WarpWS) readMeterValues(ids ...warp.Mvid) ([]float64, error) {
	result, err := w.readWSData(func() ([]float64, error) {
		result := make([]float64, len(ids))

		for i, id := range ids {
			if math.IsNaN(w.values[id]) {
				return result, api.ErrNotAvailable
			}
			result[i] = w.values[id]
		}

		return result, nil
	})

	if err != nil {
		// Return slice of correct size to allow callers to index into it.
		return make([]float64, len(ids)), err
	}

	return result, nil
}

func (w *WarpWS) currentPower() (float64, error) {
	result, err := w.readMeterValues(warp.MvidPower)
	return result[0], err
}

func (w *WarpWS) totalEnergy() (float64, error) {
	result, err := w.readMeterValues(warp.MvidEnergy)
	return result[0], err
}

func (w *WarpWS) currents() (float64, float64, float64, error) {
	result, err := w.readMeterValues(
		warp.MvidCurrentL1,
		warp.MvidCurrentL2,
		warp.MvidCurrentL3)

	return result[0], result[1], result[2], err
}

func (w *WarpWS) voltages() (float64, float64, float64, error) {
	result, err := w.readMeterValues(
		warp.MvidVoltageL1,
		warp.MvidVoltageL2,
		warp.MvidVoltageL3)

	return result[0], result[1], result[2], err
}

func (w *WarpWS) powers() (float64, float64, float64, error) {
	result, err := w.readMeterValues(
		warp.MvidPowerL1,
		warp.MvidPowerL2,
		warp.MvidPowerL3)

	return result[0], result[1], result[2], err
}

// identify reports the vehicle mac read via ISO 15118 before the RFID tag
func (w *WarpWS) identify() ([]string, error) {
	return w.readWSData(func() ([]string, error) {
		var ids []string
		// identify can be called even if the iso15118 feature is not available.
		// In that case, evState will never be written and Mac stays empty.
		// If the feature is available, but no vehicle is connected or reading
		// the MAC failed, Mac is also empty.
		// -> This handles both cases.
		if w.evState.Mac != "" {
			ids = append(ids, w.evState.Mac)
		}
		if tag := w.chargeTracker.AuthorizationInfo.TagId; tag != "" {
			ids = append(ids, tag)
		}
		return ids, nil
	})
}

// soc implements the api.Battery interface
func (w *WarpWS) soc() (float64, error) {
	return w.readWSData(func() (float64, error) {
		if w.evState.Soc != nil {
			return *w.evState.Soc, nil
		}
		return 0, api.ErrNotAvailable
	})
}

// capacity implements the api.BatteryCapacity interface
func (w *WarpWS) capacity() float64 {
	// TODO: why does api.BatteryCapacity not support returning errors?
	// Throwing it away for now, capacity will be 0 if any error occurs.
	cap, _ := w.readWSData(func() (float64, error) {
		if w.evState.Capacity != nil {
			return *w.evState.Capacity, nil
		}
		return 0, api.ErrNotAvailable
	})

	return cap
}

func (w *WarpWS) chargedEnergy() (float64, error) {
	return w.readWSData(func() (float64, error) {
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
	})
}

// ChargeDuration implements the api.ChargeTimer interface
func (w *WarpWS) ChargeDuration() (time.Duration, error) {
	return w.readWSData(func() (time.Duration, error) {
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
		var wallclock_duration = time.Since(time.Unix(int64(w.chargeTracker.TimestampMinutes)*60, 0))

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
	})
}

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (w *WarpWS) GetMinMaxCurrent() (float64, float64, error) {
	maxC, err := w.readWSData(func() (float64, error) {
		var maxCurrent = min(w.evse.Slots[0].MaxCurrent, w.evse.Slots[1].MaxCurrent)
		return float64(maxCurrent) / 1000, nil
	})

	return 6, maxC, err
}

func (w *WarpWS) callAPI(api string, payload any) error {
	var uri string
	if w.pm != w.Connection && isPmTopic(api) {
		uri = fmt.Sprintf("%s/%s", w.pm.URI, api)
	} else {
		uri = fmt.Sprintf("%s/%s", w.URI, api)
	}

	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(payload), request.JSONEncoding)
	_, err := w.Do(req)
	return err
}

func (w *WarpWS) setCurrent(curr int64) error {
	return w.callAPI("evse/external_current", curr)
}

func (w *WarpWS) disablePhaseAutoSwitch() error {
	return w.callAPI("evse/phase_auto_switch", false)
}

func (w *WarpWS) postPhasesWanted(phases int) error {
	return w.callAPI("power_manager/external_control", phases)
}

// phases1p3p implements the api.PhaseSwitcher interface
func (w *WarpWS) phases1p3p(phases int) error {
	ec, err := w.readWSData(func() (warp.ExternalControl, error) {
		return w.pmState.ExternalControl, nil
	})
	if err != nil {
		return err
	}

	// ExternalControlDeactivated is the WEM/WARP3 idle state before any
	// phases_wanted has been sent — the POST below activates external control.
	// Only block on states the POST cannot resolve.
	if ec == warp.ExternalControlRuntimeConditionsNotMet ||
		ec == warp.ExternalControlCurrentlySwitching {
		return fmt.Errorf("external control %v: %w", ec, api.ErrNotAvailable)
	}

	if err := w.postPhasesWanted(phases); err != nil {
		return err
	}
	return nil
}

// getPhases implements the api.PhaseGetter interface
func (w *WarpWS) getPhases() (int, error) {
	return w.readWSData(func() (int, error) {
		if w.pmLowLevelState.Is3phase {
			return 3, nil
		}
		return 1, nil
	})
}
