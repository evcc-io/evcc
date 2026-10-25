package tariff

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Merged combines a primary tariff with a secondary (forecast) tariff.
// Primary rates are used where available, secondary fills gaps after primary ends.
type Merged struct {
	log       *util.Logger
	primary   api.Tariff
	secondary api.Tariff
	tolerance float64 // max mean deviation of overlapping slots, 0 disables the check

	mu       sync.Mutex
	mismatch bool // last overlap check failed, logged once per change
}

// mergedMinOverlap is the minimum number of overlapping slots to compare primary and secondary
const mergedMinOverlap = 4

// NewMerged creates a merged tariff. With a positive tolerance the secondary is only appended
// while the mean absolute deviation of the slots both tariffs cover stays within the tolerance.
func NewMerged(primary, secondary api.Tariff, tolerance float64) (*Merged, error) {
	if pType, sType := primary.Type(), secondary.Type(); pType != sType {
		return nil, fmt.Errorf("primary and secondary tariff types are not compatible: %v vs %v", pType, sType)
	}

	return &Merged{
		log:       util.NewLogger("merged"),
		primary:   primary,
		secondary: secondary,
		tolerance: tolerance,
	}, nil
}

func init() {
	registry.AddCtx("merged", NewMergedFromConfig)
}

func NewMergedFromConfig(ctx context.Context, other map[string]any) (api.Tariff, error) {
	var cc struct {
		Primary, Secondary Typed
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	primary, err := NewFromConfig(ctx, cc.Primary.Type, cc.Primary.Other)
	if err != nil {
		return nil, err
	}

	secondary, err := NewFromConfig(ctx, cc.Secondary.Type, cc.Secondary.Other)
	if err != nil {
		return nil, err
	}

	return NewMerged(primary, secondary, 0)
}

// Rates implements the api.Tariff interface
func (t *Merged) Rates() (api.Rates, error) {
	result, err := t.primary.Rates()
	if err != nil {
		t.log.DEBUG.Printf("primary tariff failed, falling back to secondary: %v", err)
		return t.secondary.Rates()
	}

	secondaryRates, err := t.secondary.Rates()
	if err != nil {
		t.log.DEBUG.Printf("secondary tariff failed, using primary only: %v", err)
		return result, nil
	}

	// If primary is empty, use all secondary rates
	if len(result) == 0 {
		return secondaryRates, nil
	}

	if t.tolerance > 0 && !t.matches(result, secondaryRates) {
		return result, nil
	}

	// Find where primary data ends and append secondary rates starting there
	primaryEnd := result[len(result)-1].End
	if idx, found := slices.BinarySearchFunc(secondaryRates, primaryEnd, func(r api.Rate, t time.Time) int {
		return r.Start.Compare(t)
	}); found {
		return append(result, secondaryRates[idx:]...), nil
	}

	t.log.WARN.Printf("secondary tariff does not align gapless with primary, ignoring secondary")
	return result, nil
}

// matches checks the mean absolute deviation of the slots covered by both tariffs
func (t *Merged) matches(primary, secondary api.Rates) bool {
	values := make(map[int64]float64, len(primary))
	for _, r := range primary {
		values[r.Start.Unix()] = r.Value
	}

	var sum float64
	var n int
	for _, r := range secondary {
		if v, ok := values[r.Start.Unix()]; ok {
			sum += math.Abs(r.Value - v)
			n++
		}
	}

	mismatch := n < mergedMinOverlap || sum/float64(n) > t.tolerance

	t.mu.Lock()
	defer t.mu.Unlock()

	if mismatch != t.mismatch {
		t.mismatch = mismatch
		if mismatch {
			t.log.WARN.Printf("secondary tariff does not match primary (%d overlapping slots, mean deviation %.4f), ignoring secondary", n, sum/float64(max(n, 1)))
		} else {
			t.log.INFO.Printf("secondary tariff matches primary again (%d overlapping slots, mean deviation %.4f)", n, sum/float64(n))
		}
	}

	return !mismatch
}

// Type implements the api.Tariff interface
func (t *Merged) Type() api.TariffType {
	return t.primary.Type()
}
