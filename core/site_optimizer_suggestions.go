package core

import (
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/messenger"
	"github.com/evcc-io/evcc/tariff"
)

// evSuggestion notifies when the optimizer's advisory action for a device changes
const evSuggestion = "suggestion"

// pendingSuggestion pairs a device's current-run suggestion with the
// notification event to emit if it represents an actionable change.
type pendingSuggestion struct {
	suggestion types.Suggestion
	event      messenger.Event
}

// suggestionEvent builds the notification event for a device suggestion
func suggestionEvent(detail batteryDetail, s types.Suggestion) messenger.Event {
	ev := messenger.Event{Event: evSuggestion, Attributes: map[string]any{
		"suggestionAction": s.Action,
		"suggestionTitle":  detail.Title,
	}}

	switch {
	case detail.Type == batteryTypeBattery:
		ev.Attributes["suggestionName"] = detail.Name
	case detail.loadpoint != nil:
		id := *detail.loadpoint
		ev.Loadpoint = &id
	}

	return ev
}

// loadpointCurrentAction returns the loadpoint's current operating mode for
// suggestion comparison, reusing chargeGoalReached so a loadpoint left
// enabled while idle (e.g. vehicle finished at its limit) is treated as
// stopped instead of triggering a spurious pause suggestion.
func loadpointCurrentAction(lp *Loadpoint) string {
	lp.RLock()
	enabled := lp.enabled
	lp.RUnlock()

	if enabled && !lp.chargeGoalReached(enabled) {
		return actionCharge
	}
	return actionStop
}

// suggestionMaxAge releases a loadpoint gated by a solve the site stopped
// refreshing; the site itself expires its cached solve in reapplySuggestions
const suggestionMaxAge = 2 * tariff.SlotDuration

// setSuggestions replaces the suggestions applied on each publish
func (site *Site) setSuggestions(suggestions map[string]types.Suggestion) {
	site.Lock()
	defer site.Unlock()

	site.optimizer.suggestions = suggestions
}

// suggestion returns the optimizer suggestion for the given device key.
// The actionable flag is evaluated on read against the device's current
// action since that changes between optimizer runs.
func (site *Site) suggestion(key, currentAction string) *types.Suggestion {
	site.RLock()
	s, ok := site.optimizer.suggestions[key]
	site.RUnlock()

	if !ok {
		return nil
	}

	s.Actionable = s.Action != currentAction

	return &s
}

// publishSuggestions publishes the loadpoints' suggestions and hands them to the
// loadpoints, where they act as start/stop gate while the optimizer is in control
func (site *Site) publishSuggestions() {
	for id, lp := range site.loadpoints {
		if lp == nil {
			continue
		}

		s := site.suggestion(loadpointKey(id), loadpointCurrentAction(lp))

		var val any
		if s != nil {
			val = *s
		}
		site.publishLoadpoint(id, keys.Suggestion, val)

		lp.setSuggestion(s)
	}
}

// clearSuggestions removes all suggestions and the battery forecast when the
// optimizer result is stale
func (site *Site) clearSuggestions() {
	site.setSuggestions(nil)
	site.setBatteryForecast(nil)

	site.publishBattery()
	site.publishSuggestions()

	site.Lock()
	site.optimizer.suggestionActions = nil
	site.optimizer.lastSolve = nil
	site.Unlock()
}

// pendingSuggestions collects the stored suggestions with their actionable flag
// evaluated against the devices' current operating mode
func (site *Site) pendingSuggestions(details []batteryDetail) map[string]pendingSuggestion {
	pending := make(map[string]pendingSuggestion, len(details))

	for _, detail := range details {
		key := detail.key()
		if key == "" {
			continue
		}

		s := site.suggestion(key, detail.currentAction(site))
		if s == nil {
			continue
		}

		pending[key] = pendingSuggestion{suggestion: *s, event: suggestionEvent(detail, *s)}
	}

	return pending
}

// diffSuggestions updates the tracked actionable optimizer suggestions and
// returns the events to send for devices whose actionable action changed since
// the last run. Non-actionable or vanished devices are pruned so a later
// actionable change re-notifies.
func (site *Site) diffSuggestions(pending map[string]pendingSuggestion) []messenger.Event {
	site.Lock()
	defer site.Unlock()

	if site.optimizer.suggestionActions == nil {
		site.optimizer.suggestionActions = make(map[string]string)
	}

	// prune devices that are gone or no longer actionable
	for key := range site.optimizer.suggestionActions {
		if p, ok := pending[key]; !ok || !p.suggestion.Actionable {
			delete(site.optimizer.suggestionActions, key)
		}
	}

	var events []messenger.Event
	for key, p := range pending {
		if !p.suggestion.Actionable || site.optimizer.suggestionActions[key] == p.suggestion.Action {
			continue
		}
		site.optimizer.suggestionActions[key] = p.suggestion.Action
		events = append(events, p.event)
	}
	return events
}
