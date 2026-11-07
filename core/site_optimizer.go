package core

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	optimizer "github.com/evcc-io/optimizer/client"
)

const (
	// eta is the efficiency of the battery charging/discharging
	eta = 0.9

	// batteryPower is the default power of the battery in W
	batteryPower = 6000
)

// optimizerChargingStrategies are the valid grid charging strategies; the first
// entry is the default and preserves the previous hard-coded behavior.
var optimizerChargingStrategies = []string{
	string(optimizer.OptimizerStrategyChargingStrategyChargeBeforeExport),
	string(optimizer.OptimizerStrategyChargingStrategyAttenuateDemandPeaks),
	string(optimizer.OptimizerStrategyChargingStrategyAttenuateFeedinPeaks),
	string(optimizer.OptimizerStrategyChargingStrategyAttenuateGridPeaks),
	string(optimizer.OptimizerStrategyChargingStrategyNone),
}

const defaultOptimizerChargingStrategy = string(optimizer.OptimizerStrategyChargingStrategyChargeBeforeExport)

// optimizerResult wraps the optimizer publish payload to implement BytesMarshaler.
// This ensures publishComplex serializes it as a single JSON message instead of
// recursively decomposing each struct field and array element into individual MQTT
// topics (~1,500 messages per optimizer run).
type optimizerResult struct {
	Updated time.Time                    `json:"updated"`
	Req     optimizer.OptimizationInput  `json:"req"`
	Res     optimizer.OptimizationResult `json:"res"`
	Details requestDetails               `json:"details"`
	Hosted  bool                         `json:"hosted"` // explain needs the hosted optimizer
}

var _ api.BytesMarshaler = (*optimizerResult)(nil)

func (r optimizerResult) MarshalBytes() ([]byte, error) {
	return json.Marshal(r)
}

type batteryType string

const (
	OPTIMIZER_URI = "https://optimizer.evcc.io"

	batteryTypeLoadpoint batteryType = "loadpoint"
	batteryTypeVehicle   batteryType = "vehicle"
	batteryTypeBattery   batteryType = "battery"
)

type batteryDetail struct {
	Type     batteryType `json:"type"`
	Title    string      `json:"title,omitempty"`
	Name     string      `json:"name,omitempty"`
	Capacity float64     `json:"capacity,omitempty"`

	loadpoint    *int              // originating loadpoint id for loadpoint/vehicle entries
	controllable bool              // device can act on suggestions
	modes        []api.BatteryMode // modes the battery controller supports
}

// batteryKey and loadpointKey build the canonical device keys used for
// suggestion routing and notifications
func batteryKey(name string) string { return "battery:" + name }
func loadpointKey(id int) string    { return fmt.Sprintf("loadpoint:%d", id) }

// key identifies the device across optimizer runs; an empty key means the
// device can't act on a suggestion.
func (d batteryDetail) key() string {
	switch {
	case d.Type == batteryTypeBattery:
		return batteryKey(d.Name)
	case d.loadpoint != nil:
		return loadpointKey(*d.loadpoint)
	default:
		return ""
	}
}

// currentAction returns the device's current operating mode for suggestion
// comparison. Must only be called for devices with a non-empty key.
func (d batteryDetail) currentAction(site *Site) string {
	if d.Type == batteryTypeBattery {
		return site.batteryAction()
	}
	return loadpointCurrentAction(site.loadpoints[*d.loadpoint])
}

// batteryAction returns the battery's current mode for suggestion comparison.
// A battery that was never switched (BatteryUnknown) is in normal operation.
func (site *Site) batteryAction() string {
	if mode := site.GetBatteryMode(); mode != api.BatteryUnknown {
		return mode.String()
	}
	return api.BatteryNormal.String()
}

func optimizerURI() string {
	return cmp.Or(os.Getenv("OPTIMIZER_URI"), OPTIMIZER_URI)
}

// optimizerAuth adds the sponsor token for the hosted optimizer only, a custom OPTIMIZER_URI must not receive it
func optimizerAuth(req *http.Request) {
	if optimizerURI() == OPTIMIZER_URI && sponsor.IsAuthorizedForApi() {
		req.Header.Set("Authorization", "Bearer "+sponsor.Token())
	}
}

// errOptimizerNotReady means battery measurements aren't available yet (e.g. at
// startup); the slot gate is left open so the next cycle retries.
var errOptimizerNotReady = errors.New("battery measurements not ready")

// optimizerInterval is the refresh cadence in automatic mode. It divides the
// slot duration so every slot starts on a fresh result. Advisory suggestions
// only need one run per slot.
const optimizerInterval = 5 * time.Minute

// optimizerUpdateAsync runs the optimizer unless the last run is younger than
// the mode's interval. Pass force to run regardless, e.g. when a changed setting
// should take effect immediately. It is a no-op when the optimizer is not
// active. A forced trigger during a running update re-runs once that update
// has finished.
func (site *Site) optimizerUpdateAsync(force bool) {
	if !sponsor.IsAuthorized() || !optimizerEnabled() {
		return
	}

	// store before locking: either we get the lock, or the holder sees the trigger on unlock
	if force {
		site.optimizer.rerun.Store(true)
	}

	if !site.optimizer.mu.TryLock() {
		return
	}
	defer site.optimizerUnlock()

	// false if a run that started after the trigger already served it
	force = site.optimizer.rerun.Swap(false)

	interval := tariff.SlotDuration
	if site.Automatic() {
		interval = optimizerInterval
	}

	if force {
		// keep the gate open so a not-ready run is retried on the next cycle
		site.optimizer.updated = time.Time{}
	} else if time.Since(site.optimizer.updated) < interval {
		return
	}

	var err error

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic %v", r)
		}

		// not ready yet: keep the gate open for an immediate retry next cycle
		if errors.Is(err, errOptimizerNotReady) {
			return
		}

		site.optimizer.updated = time.Now()

		// a failed run keeps the advice: dropping it would release the controlled
		// devices for one cycle. reapplySuggestions expires the cached solve.
		if err != nil {
			site.log.ERROR.Println("optimizer:", err)
		}
	}()

	err = site.optimizerUpdate(site.state().battery.Devices)
}

func (site *Site) optimizerUpdate(battery []types.Measurement) error {
	req, details, err := site.optimizerRequest(battery)
	if err != nil {
		return err
	}

	if len(req.Batteries) == 0 {
		// meters configured but measurements not in yet: retry instead of
		// consuming the slot gate
		if len(site.batteryMeters) > 0 {
			return errOptimizerNotReady
		}
		return nil // nothing to optimize
	}

	httpClient := request.NewClient(site.log)
	httpClient.Timeout = 90 * time.Second

	apiClient, err := optimizer.NewClientWithResponses(optimizerURI(), optimizer.WithHTTPClient(httpClient))
	if err != nil {
		return err
	}

	resp, err := apiClient.PostOptimizeChargeScheduleWithResponse(context.TODO(), req, func(_ context.Context, req *http.Request) error {
		optimizerAuth(req)
		return nil
	})
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return apiError(resp)
	}

	// publish before the status check so the optimizer page stays available
	// for diagnosing non-optimal results
	result := optimizerResult{
		Updated: time.Now(),
		Req:     req,
		Res:     *resp.JSON200,
		Details: details,
		Hosted:  optimizerURI() == OPTIMIZER_URI,
	}

	site.Lock()
	site.optimizer.result = &result
	site.Unlock()

	site.publish("evopt", result)

	// feasible results are usable, they are just not proven optimal
	if status := resp.JSON200.Status; status != optimizer.Optimal && status != optimizer.Feasible {
		return errors.New(string(status))
	}

	if len(details.Timestamps) != len(req.TimeSeries.Dt) || len(req.Batteries) != len(resp.JSON200.Batteries) {
		return errors.New("inconsistent optimizer result dimensions")
	}

	now := time.Now()
	schedule := optimizerSchedule{timestamps: details.Timestamps, dt: req.TimeSeries.Dt}
	if schedule.activeSlot(now) < 0 {
		return errors.New("optimizer result expired")
	}

	site.applyOptimizerResult(optimizerSolve{req: req, details: details, res: *resp.JSON200, schedule: schedule, completed: now}, now)

	return nil
}

// optimizerState holds the optimizer runtime state of a site
type optimizerState struct {
	mu      sync.Mutex  // guards optimizer runs
	updated time.Time   // last optimizer run, guarded by mu
	rerun   atomic.Bool // forced trigger arrived during a run, re-run when done

	// guarded by the site lock
	suggestions       map[string]types.Suggestion // suggestions by device key
	suggestionActions map[string]string           // last notified actionable action by device key
	lastSolve         *optimizerSolve             // last successful solve, reapplied to newer slots by the control cycle
	result            *optimizerResult            // last published plan, explained on request

	explainMu sync.Mutex   // serializes explain requests
	explained explainCache // last explanation, guarded by explainMu
}

// optimizerSolve caches a solve so the control cycle can reapply it to a newer slot
type optimizerSolve struct {
	req       optimizer.OptimizationInput
	details   requestDetails
	res       optimizer.OptimizationResult
	schedule  optimizerSchedule
	slot      int       // last applied slot
	completed time.Time // solve completion, not last reapply
}

// setLastOptimizerSolve remembers a solve's inputs for reapplySuggestions
func (site *Site) setLastOptimizerSolve(solve *optimizerSolve) {
	site.Lock()
	defer site.Unlock()

	site.optimizer.lastSolve = solve
}

// optimizerUnlock releases optimizer.mu and re-runs for a forced trigger that arrived while it was held
func (site *Site) optimizerUnlock() {
	site.optimizer.mu.Unlock()

	if site.optimizer.rerun.Load() {
		go site.optimizerUpdateAsync(false)
	}
}

// reapplySuggestions re-derives the last solve's suggestions for the slot covering now
// without a new solve. TryLock so it never overwrites a fresher concurrent solve.
func (site *Site) reapplySuggestions(now time.Time) {
	if !site.optimizer.mu.TryLock() {
		return
	}
	defer site.optimizerUnlock()

	site.RLock()
	last := site.optimizer.lastSolve
	site.RUnlock()

	if last == nil {
		return
	}

	slot := last.schedule.activeSlot(now)
	if slot == last.slot {
		return
	}

	// horizon passed or no completed solve for two slots: don't march a dead plan forward
	if slot < 0 || now.Sub(last.completed) > 2*tariff.SlotDuration {
		site.log.DEBUG.Println("optimizer: cached result expired")
		site.clearSuggestions()
		return
	}

	// disconnected loadpoints are excluded from a fresh solve; don't advise an empty charger
	for _, d := range last.details.BatteryDetails {
		if d.loadpoint == nil {
			continue
		}
		if lp := site.loadpoints[*d.loadpoint]; lp == nil || !lp.connected() {
			site.setLastOptimizerSolve(nil)
			return
		}
	}

	site.applyOptimizerResult(*last, now)
}

// apiError extracts error message from optimizer API response
func apiError(resp *optimizer.PostOptimizeChargeScheduleResponse) error {
	var errObj *optimizer.Error
	switch resp.StatusCode() {
	case http.StatusBadRequest:
		errObj = resp.JSON400
	case http.StatusInternalServerError:
		errObj = resp.JSON500
	}

	if errObj == nil {
		return fmt.Errorf("invalid status: %d: %s", resp.StatusCode(), resp.Body)
	}

	if len(errObj.Details) > 0 {
		var details []string
		for field, msg := range errObj.Details {
			details = append(details, fmt.Sprintf("%s: %s", field, msg))
		}
		slices.Sort(details)
		return fmt.Errorf("%s (%s)", errObj.Message, strings.Join(details, ", "))
	}

	return errors.New(errObj.Message)
}
