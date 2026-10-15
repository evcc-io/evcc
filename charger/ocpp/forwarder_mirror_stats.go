package ocpp

// Statistics of what was written to each mirror's upstream, per hour. They let the
// owner compare what evcc reported with what the upstream received.

import (
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
)

const (
	mirrorHistoryHours = 72
	mirrorStatusHours  = 48

	// a larger step of the energy register is a reset or a gap, not delivered energy
	maxEnergyStepWh = 100_000
)

// MirrorHour counts what was written to a mirror's upstream within one hour.
type MirrorHour struct {
	Start       time.Time `json:"start"`
	Messages    int       `json:"messages"`
	MeterValues int       `json:"meterValues"`
	EnergyWh    float64   `json:"energyWh"`
	Disconnects int       `json:"disconnects"`
}

type mirrorStats struct {
	Hours    []MirrorHour `json:"hours"`
	LastWh   float64      `json:"lastWh,omitempty"`
	LastSent time.Time    `json:"lastSent,omitzero"`
}

// mirrorStatsStore persists the statistics of all mirrors
type mirrorStatsStore interface {
	load() (map[string]mirrorStats, error)
	save(map[string]mirrorStats) error
}

// mirrorStatsBackend is the store in use; tests replace it
var mirrorStatsBackend mirrorStatsStore = settingsMirrorStats{}

type settingsMirrorStats struct{}

func (settingsMirrorStats) load() (map[string]mirrorStats, error) {
	var res map[string]mirrorStats
	if err := settings.Json(keys.OcppMirrorStats, &res); err != nil {
		if errors.Is(err, settings.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (settingsMirrorStats) save(stats map[string]mirrorStats) error {
	return settings.SetJson(keys.OcppMirrorStats, stats)
}

// guarded by mirrorsMu
var (
	mirrorStatsData   = make(map[string]*mirrorStats)
	mirrorStatsLoaded bool
	mirrorStatsDirty  bool

	// set when the stored statistics could not be read: what is recorded stays in memory
	// and the stored data is left alone instead of being replaced by a fraction of it
	mirrorStatsReadOnly bool
)

// serializes saving, so that an older snapshot cannot overwrite a newer one
var mirrorStatsSaveMu sync.Mutex

func mirrorStatsKey(chargerID, mirrorKey string) string {
	return chargerID + "|" + mirrorKey
}

// loadStatsLocked reads the stored statistics once. Caller holds mirrorsMu.
func loadStatsLocked() {
	if mirrorStatsLoaded {
		return
	}
	mirrorStatsLoaded = true

	stored, err := mirrorStatsBackend.load()
	if err != nil {
		mirrorStatsReadOnly = true
		forwarderLog.ERROR.Printf("mirror: read statistics, keeping stored data untouched: %v", err)
		return
	}

	for k, v := range stored {
		mirrorStatsData[k] = &mirrorStats{Hours: v.Hours, LastWh: v.LastWh, LastSent: v.LastSent}
	}
}

// statsLocked returns the statistics of a mirror. Caller holds mirrorsMu.
func statsLocked(key string) *mirrorStats {
	loadStatsLocked()

	st := mirrorStatsData[key]
	if st == nil {
		st = new(mirrorStats)
		mirrorStatsData[key] = st
	}
	return st
}

// hourLocked returns the bucket of the hour containing now. Starting a new hour
// writes the finished one to the log. Caller holds mirrorsMu.
func hourLocked(st *mirrorStats, chargerID string, now time.Time) *MirrorHour {
	start := now.Truncate(time.Hour)

	if n := len(st.Hours); n > 0 {
		if st.Hours[n-1].Start.Equal(start) {
			return &st.Hours[n-1]
		}

		h := st.Hours[n-1]
		forwarderLog.INFO.Printf("mirror %s: %s: %d messages (%d meter values), %.3f kWh, %d disconnects",
			chargerID, h.Start.Local().Format("02 Jan 15h"), h.Messages, h.MeterValues, h.EnergyWh/1e3, h.Disconnects)
	}

	st.Hours = append(st.Hours, MirrorHour{Start: start})
	if len(st.Hours) > mirrorHistoryHours {
		st.Hours = st.Hours[len(st.Hours)-mirrorHistoryHours:]
	}

	return &st.Hours[len(st.Hours)-1]
}

// record counts a frame that was written to the mirror's upstream.
func (ms *mirrorSession) record(frame []byte, now time.Time) {
	_, _, action, err := parseOCPPFrame(frame)
	isMeter := err == nil && action == "MeterValues"

	var wh float64
	var hasEnergy bool
	if isMeter {
		wh, hasEnergy = meterEnergyWh(frame)
	}

	mirrorsMu.Lock()
	defer mirrorsMu.Unlock()

	st := statsLocked(mirrorStatsKey(ms.chargerID, ms.rule.Key()))
	h := hourLocked(st, ms.chargerID, now)

	h.Messages++
	if isMeter {
		h.MeterValues++
	}
	if hasEnergy {
		if step := wh - st.LastWh; st.LastWh > 0 && step >= 0 && step <= maxEnergyStepWh {
			h.EnergyWh += step
		}
		st.LastWh = wh
	}

	st.LastSent = now
	mirrorStatsDirty = true
}

// recordDisconnectLocked counts a lost connection. Caller holds mirrorsMu.
func (ms *mirrorSession) recordDisconnectLocked(now time.Time) {
	st := statsLocked(mirrorStatsKey(ms.chargerID, ms.rule.Key()))
	hourLocked(st, ms.chargerID, now).Disconnects++
	mirrorStatsDirty = true
}

// meterEnergyWh returns the active import energy register of a MeterValues frame in Wh.
func meterEnergyWh(frame []byte) (float64, bool) {
	var msg []json.RawMessage
	if json.Unmarshal(frame, &msg) != nil || len(msg) < 4 {
		return 0, false
	}

	var payload struct {
		MeterValue []struct {
			SampledValue []struct {
				Value     string `json:"value"`
				Measurand string `json:"measurand"`
				Unit      string `json:"unit"`
				Phase     string `json:"phase"`
			} `json:"sampledValue"`
		} `json:"meterValue"`
	}
	if json.Unmarshal(msg[3], &payload) != nil {
		return 0, false
	}

	for _, mv := range payload.MeterValue {
		for _, sv := range mv.SampledValue {
			if sv.Phase != "" || (sv.Measurand != "" && sv.Measurand != "Energy.Active.Import.Register") {
				continue
			}

			f, err := strconv.ParseFloat(sv.Value, 64)
			if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
				continue
			}
			if sv.Unit == "kWh" {
				f *= 1e3
			}
			return f, true
		}
	}

	return 0, false
}

// mirrorListed reports whether the rule of a charger lists a mirror. Caller holds mirrorsMu.
func mirrorListed(chargerID, mirrorKey string) bool {
	rule, ok := resolveRule(chargerID)
	if !ok {
		return false
	}

	return slices.ContainsFunc(rule.Mirrors, func(m MirrorRule) bool { return m.Key() == mirrorKey })
}

// pruneMirrorStats drops the statistics of mirrors that no rule lists anymore, whether or
// not their charger is connected, so a mirror that is added again starts empty.
func pruneMirrorStats() {
	mirrorsMu.Lock()
	defer mirrorsMu.Unlock()

	loadStatsLocked()

	for key := range mirrorStatsData {
		chargerID, mirrorKey, ok := strings.Cut(key, "|")
		if !ok || !mirrorListed(chargerID, mirrorKey) {
			delete(mirrorStatsData, key)
			mirrorStatsDirty = true
		}
	}
}

// flushMirrorStats persists the statistics and republishes the status if anything changed.
// A failed save keeps the statistics dirty so the next flush retries it.
func flushMirrorStats() {
	mirrorStatsSaveMu.Lock()
	saved := saveMirrorStats()
	mirrorStatsSaveMu.Unlock()

	if saved {
		notifyUpdated()
	}
}

func saveMirrorStats() bool {
	mirrorsMu.Lock()
	if !mirrorStatsDirty || mirrorStatsReadOnly {
		mirrorsMu.Unlock()
		return false
	}
	mirrorStatsDirty = false

	snapshot := make(map[string]mirrorStats, len(mirrorStatsData))
	for k, v := range mirrorStatsData {
		snapshot[k] = mirrorStats{Hours: slices.Clone(v.Hours), LastWh: v.LastWh, LastSent: v.LastSent}
	}
	mirrorsMu.Unlock()

	if err := mirrorStatsBackend.save(snapshot); err != nil {
		forwarderLog.ERROR.Printf("mirror: save statistics: %v", err)

		mirrorsMu.Lock()
		mirrorStatsDirty = true
		mirrorsMu.Unlock()
		return false
	}

	return true
}
