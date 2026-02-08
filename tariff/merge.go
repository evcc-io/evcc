package tariff

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Merge combines a primary tariff with a secondary (forecast) tariff.
// Primary rates are used where available, secondary fills gaps after primary ends.
type Merge struct {
	log       *util.Logger
	primary   api.Tariff
	secondary api.Tariff
}

func init() {
	registry.AddCtx("merge", NewMergeFromConfig)
}

func NewMergeFromConfig(ctx context.Context, other map[string]any) (api.Tariff, error) {
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

	if pType, sType := primary.Type(), secondary.Type(); pType != sType {
		return nil, fmt.Errorf("primary and secondary tariff types are not compatible: %v vs %v", pType, sType)
	}

	t := &Merge{
		log:       util.NewLogger("merge"),
		primary:   primary,
		secondary: secondary,
	}

	return t, nil
}

// Rates implements the api.Tariff interface
func (t *Merge) Rates() (api.Rates, error) {
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

	// Find where primary data ends and append secondary rates starting there
	primaryEnd := result[len(result)-1].End
	if idx, ok := slices.BinarySearchFunc(secondaryRates, primaryEnd, func(r api.Rate, t time.Time) int {
		return r.Start.Compare(t)
	}); ok {
		return append(result, secondaryRates[idx:]...), nil
	}

	t.log.WARN.Printf("secondary tariff does not align gaplessly with primary, ignoring secondary")
	return result, nil
}

// Type implements the api.Tariff interface
func (t *Merge) Type() api.TariffType {
	return t.primary.Type()
}
