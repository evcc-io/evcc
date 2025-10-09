package planner

import (
	"slices"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Planner plans a series of charging slots for a given (variable) tariff
type Planner struct {
	log    *util.Logger
	clock  clock.Clock // mockable time
	tariff api.Tariff
}

// New creates a price planner
func New(log *util.Logger, tariff api.Tariff, opt ...func(t *Planner)) *Planner {
	p := &Planner{
		log:    log,
		clock:  clock.New(),
		tariff: tariff,
	}

	for _, o := range opt {
		o(p)
	}

	return p
}

// Configuration for charge planning optimization
const (
	// InterruptionPenaltyPercent is the cost penalty for fragmenting charging sessions
	// Applied as percentage of average cost per interruption (gap between windows)
	// Example: 0.05 means 5% penalty, so fragmentation only occurs if it saves >5% per gap
	// With typical charging (11-22 kW, 2-4h): Fragmentation only if saves ~0.50-1.50 €
	InterruptionPenaltyPercent = 0.05
)

// chargingWindow represents a continuous block of charging slots
type chargingWindow struct {
	slots    api.Rates
	start    time.Time
	end      time.Time
	duration time.Duration
	avgCost  float64
}

// planCandidate represents a potential charging plan
type planCandidate struct {
	windows   []chargingWindow
	totalCost float64
	score     float64 // lower is better (pure average cost)
	plan      api.Rates
}

// filterValidSlots filters and adjusts slots to the valid time range
func (t *Planner) filterValidSlots(rates api.Rates, targetTime time.Time) api.Rates {
	var validSlots api.Rates

	for _, rate := range rates {
		// Skip slots outside valid time range (updated logic from latest version)
		if !(rate.End.After(t.clock.Now()) && rate.Start.Before(targetTime)) {
			continue
		}

		// Adjust slot boundaries
		slot := rate
		if slot.Start.Before(t.clock.Now()) {
			slot.Start = t.clock.Now()
		}
		if slot.End.After(targetTime) {
			slot.End = targetTime
		}

		validSlots = append(validSlots, slot)
	}

	return validSlots
}

// generateChargingWindows creates all possible continuous charging windows
// AND individual slots as single-slot windows
func (t *Planner) generateChargingWindows(slots api.Rates) []chargingWindow {
	if len(slots) == 0 {
		return nil
	}

	// Sort slots by start time
	slices.SortFunc(slots, func(a, b api.Rate) int {
		return a.Start.Compare(b.Start)
	})

	var windows []chargingWindow

	// Generate windows of different lengths starting from each position
	for i := 0; i < len(slots); i++ {
		window := chargingWindow{
			start: slots[i].Start,
		}

		for j := i; j < len(slots); j++ {
			// Check if slots are consecutive
			if j > i {
				prevSlot := slots[j-1]
				currSlot := slots[j]

				// If there's a gap, break this window
				if !currSlot.Start.Equal(prevSlot.End) {
					break
				}
			}

			// Add slot to window
			window.slots = append(window.slots, slots[j])
			window.end = slots[j].End
			window.duration = window.end.Sub(window.start)

			// Calculate average cost for this window
			window.avgCost = AverageCost(window.slots)

			// Always add the window (including single-slot windows)
			windows = append(windows, window)
		}
	}

	return windows
}

// findBestWindowCombination finds the optimal combination of windows
func (t *Planner) findBestWindowCombination(windows []chargingWindow, requiredDuration time.Duration) *planCandidate {
	if len(windows) == 0 {
		return nil
	}

	// Sort windows by a composite score that favors:
	// 1. Lower cost (primary)
	// 2. Longer duration (secondary - for hardware protection)
	// 3. Later start time (tertiary - original behavior)
	slices.SortFunc(windows, func(a, b chargingWindow) int {
		// Compare costs with a small tolerance to allow duration preference
		costDiff := a.avgCost - b.avgCost
		const costTolerance = 0.001 // Very small tolerance for floating point comparison

		if costDiff < -costTolerance {
			return -1 // a is significantly cheaper
		}
		if costDiff > costTolerance {
			return 1 // b is significantly cheaper
		}

		// Costs are essentially equal - prefer longer duration (hardware friendly)
		if a.duration > b.duration {
			return -1
		}
		if a.duration < b.duration {
			return 1
		}

		// Same cost and duration: prefer later start time (original behavior)
		return b.start.Compare(a.start)
	})

	// Try to find the best combination by evaluating different strategies:
	// 1. Single continuous window (preferred for hardware)
	// 2. Multiple windows with lowest total cost

	var bestCandidate *planCandidate

	// Strategy 1: Try single continuous window first (best for hardware)
	for i := range windows {
		w := &windows[i]
		if w.duration >= requiredDuration {
			candidate := t.evaluateWindowCombination([]chargingWindow{*w}, requiredDuration)
			if candidate != nil {
				// Score favors single windows: pure cost
				if bestCandidate == nil || candidate.score < bestCandidate.score {
					bestCandidate = candidate
				}
			}
		}
	}

	// Strategy 2: Greedy selection of multiple windows
	// Only consider this if no single window was found or if it could be cheaper
	var selected []chargingWindow
	var totalDuration time.Duration

	for _, w := range windows {
		// Check if this window overlaps with already selected ones
		overlaps := false
		for _, sel := range selected {
			if w.start.Before(sel.end) && w.end.After(sel.start) {
				overlaps = true
				break
			}
		}

		if overlaps {
			continue
		}

		// Add this window
		selected = append(selected, w)
		totalDuration += w.duration

		// Stop if we have enough duration
		if totalDuration >= requiredDuration {
			break
		}
	}

	// Evaluate the multi-window combination
	if len(selected) > 0 && (bestCandidate == nil || len(selected) > 1) {
		candidate := t.evaluateWindowCombination(selected, requiredDuration)
		if candidate != nil {
			// Apply interruption penalty to favor continuous charging
			// This balances hardware protection with cost efficiency
			interruptionPenalty := candidate.score * InterruptionPenaltyPercent * float64(len(candidate.windows)-1)
			score := candidate.score + interruptionPenalty

			if bestCandidate == nil || score < bestCandidate.score {
				bestCandidate = candidate
			}
		}
	}

	return bestCandidate
}

// evaluateWindowCombination calculates the score for a window combination
func (t *Planner) evaluateWindowCombination(windows []chargingWindow, requiredDuration time.Duration) *planCandidate {
	if len(windows) == 0 {
		return nil
	}

	// Sort windows by start time
	slices.SortFunc(windows, func(a, b chargingWindow) int {
		return a.start.Compare(b.start)
	})

	// Calculate total duration and ensure we have enough
	var totalDuration time.Duration
	var allSlots api.Rates
	for _, w := range windows {
		totalDuration += w.duration
		allSlots = append(allSlots, w.slots...)
	}

	// Check if we have enough duration to meet the requirement
	if totalDuration < requiredDuration {
		return nil
	}

	// Adjust if we have too much duration - using original logic
	if totalDuration > requiredDuration {
		excess := totalDuration - requiredDuration

		// Apply original shortening logic:
		// - First (but not single) window: shift start forward (late start)
		// - Otherwise: shift last window's end backward (early end)

		if len(windows) > 1 {
			// Multiple windows: adjust first window's start
			firstWindow := &windows[0]
			if firstWindow.duration > excess {
				firstWindow.start = firstWindow.start.Add(excess)
				firstWindow.duration -= excess

				// Adjust slots in first window
				var adjustedSlots api.Rates
				for _, slot := range firstWindow.slots {
					if slot.End.After(firstWindow.start) {
						adjustedSlot := slot
						if adjustedSlot.Start.Before(firstWindow.start) {
							adjustedSlot.Start = firstWindow.start
						}
						adjustedSlots = append(adjustedSlots, adjustedSlot)
					}
				}
				firstWindow.slots = adjustedSlots

				// Recalculate average cost
				if len(firstWindow.slots) > 0 {
					firstWindow.avgCost = AverageCost(firstWindow.slots)
				}
			} else {
				// If first window is too short, we need to remove it and shorten the next
				// This shouldn't happen with correct window selection
				return nil
			}
		} else {
			// Single window: adjust end
			lastWindow := &windows[0]
			if lastWindow.duration > excess {
				lastWindow.duration -= excess
				lastWindow.end = lastWindow.end.Add(-excess)

				// Remove excess slots from the end
				var adjustedSlots api.Rates
				for _, slot := range lastWindow.slots {
					if slot.Start.Before(lastWindow.end) {
						adjustedSlot := slot
						if adjustedSlot.End.After(lastWindow.end) {
							adjustedSlot.End = lastWindow.end
						}
						adjustedSlots = append(adjustedSlots, adjustedSlot)
					}
				}
				lastWindow.slots = adjustedSlots

				// Recalculate average cost
				if len(lastWindow.slots) > 0 {
					lastWindow.avgCost = AverageCost(lastWindow.slots)
				}
			} else {
				return nil
			}
		}

		// Rebuild allSlots after adjustment
		allSlots = allSlots[:0]
		totalDuration = 0
		for _, w := range windows {
			allSlots = append(allSlots, w.slots...)
			totalDuration += w.duration
		}
	}

	// Calculate weighted average cost across all windows
	var totalCost float64
	for _, w := range windows {
		totalCost += w.avgCost * float64(w.duration)
	}

	return &planCandidate{
		windows:   windows,
		totalCost: totalCost,
		score:     totalCost / float64(totalDuration),
		plan:      allSlots,
	}
}

// plan creates a lowest-cost plan using window bundling optimization
func (t *Planner) plan(rates api.Rates, requiredDuration time.Duration, targetTime time.Time) api.Rates {
	// Filter and adjust slots to valid time range
	validSlots := t.filterValidSlots(rates, targetTime)
	if len(validSlots) == 0 {
		return nil
	}

	// Generate all possible charging windows
	windows := t.generateChargingWindows(validSlots)
	if len(windows) == 0 {
		return nil
	}

	// Find best combination of windows
	bestCandidate := t.findBestWindowCombination(windows, requiredDuration)
	if bestCandidate == nil {
		return nil
	}

	return bestCandidate.plan
}

// continuousPlan creates a continuous emergency charging plan
func (t *Planner) continuousPlan(rates api.Rates, start, end time.Time) api.Rates {
	rates.Sort()

	res := make(api.Rates, 0, len(rates)+2)
	for _, r := range rates {
		// slot before continuous plan
		if !r.End.After(start) {
			continue
		}

		// slot after continuous plan
		if !r.Start.Before(end) {
			continue
		}

		// adjust first slot
		if r.Start.Before(start) && r.End.After(start) {
			r.Start = start
		}

		// adjust last slot
		if r.Start.Before(end) && r.End.After(end) {
			r.End = end
		}

		res = append(res, r)
	}

	if len(res) == 0 {
		res = append(res, api.Rate{
			Start: start,
			End:   end,
		})
	} else {
		// prepend missing slot
		if res[0].Start.After(start) {
			res = slices.Insert(res, 0, api.Rate{
				Start: start,
				End:   res[0].Start,
			})
		}
		// append missing slot
		if last := res[len(res)-1]; last.End.Before(end) {
			res = append(res, api.Rate{
				Start: last.End,
				End:   end,
			})
		}
	}

	return res
}

func (t *Planner) Plan(requiredDuration, precondition time.Duration, targetTime time.Time) api.Rates {
	if t == nil || requiredDuration <= 0 {
		return nil
	}

	latestStart := targetTime.Add(-requiredDuration)
	if latestStart.Before(t.clock.Now()) {
		latestStart = t.clock.Now()
		targetTime = latestStart.Add(requiredDuration)
	}

	// simplePlan only considers time, but not cost
	simplePlan := api.Rates{
		api.Rate{
			Start: latestStart,
			End:   targetTime,
		},
	}

	// target charging without tariff or late start
	if t.tariff == nil {
		return simplePlan
	}

	rates, err := t.tariff.Rates()

	// treat like normal target charging if we don't have rates
	if len(rates) == 0 || err != nil {
		return simplePlan
	}

	// consume remaining time
	if t.clock.Until(targetTime) <= requiredDuration {
		return t.continuousPlan(rates, latestStart, targetTime)
	}

	// rates are by default sorted by date, oldest to newest
	last := rates[len(rates)-1].End

	// Two-phase approach for precondition:
	// Phase 1: Extract and reserve precondition slots (separate from optimization)
	// Phase 2: Optimize remaining duration with window bundling

	var preconditionPlan api.Rates
	var remainingDuration = requiredDuration
	var optimizationEnd = targetTime

	if precondition > 0 {
		// Precondition zone should not exceed required duration
		// Example: If need 2h and precondition is "all" (10h), only mark last 2h
		effectivePrecondition := precondition
		if effectivePrecondition > requiredDuration {
			effectivePrecondition = requiredDuration
		}

		preCondStart := targetTime.Add(-effectivePrecondition)

		// Extract precondition slots - these are handled separately
		// to prevent them from being merged into optimization windows
		for _, r := range rates {
			if r.End.After(preCondStart) && r.Start.Before(targetTime) {
				slot := r

				// Adjust to precondition boundaries
				if slot.Start.Before(preCondStart) {
					slot.Start = preCondStart
				}
				if slot.End.After(targetTime) {
					slot.End = targetTime
				}

				preconditionPlan = append(preconditionPlan, slot)
				slotDuration := slot.End.Sub(slot.Start)
				remainingDuration -= slotDuration
			}
		}

		// Adjust optimization window to exclude precondition zone
		optimizationEnd = preCondStart

		// If precondition covers all or more than required duration
		if remainingDuration <= 0 {
			if remainingDuration < 0 {
				// Need to shorten precondition plan
				excess := -remainingDuration
				if len(preconditionPlan) > 0 {
					preconditionPlan[0].Start = preconditionPlan[0].Start.Add(excess)
				}
			}
			preconditionPlan.Sort()
			return preconditionPlan
		}
	}

	// Now optimize the remaining duration BEFORE precondition zone
	// This prevents precondition slots from affecting window optimization

	// sort rates by price and time
	slices.SortStableFunc(rates, sortByCost)

	// reduce planning horizon to available rates
	if optimizationEnd.After(last) {
		durationAfterRates := optimizationEnd.Sub(last)
		if durationAfterRates >= remainingDuration {
			// All remaining can be charged after known rates
			if len(preconditionPlan) == 0 {
				return nil
			}
			preconditionPlan.Sort()
			return preconditionPlan
		}

		t.log.DEBUG.Printf("target time beyond available slots- reducing plan horizon from %v to %v",
			remainingDuration.Round(time.Second), (remainingDuration - durationAfterRates).Round(time.Second))

		optimizationEnd = last
		remainingDuration -= durationAfterRates
	}

	// Get optimized plan for remaining duration (excludes precondition zone)
	optimizedPlan := t.plan(rates, remainingDuration, optimizationEnd)

	// Combine: optimized main charging + mandatory precondition
	combinedPlan := append(optimizedPlan, preconditionPlan...)
	combinedPlan.Sort()

	return combinedPlan
}
