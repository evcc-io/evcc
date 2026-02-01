package tariff

import (
	"context"
	"fmt"
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
}

func init() {
	registry.AddCtx("merged", NewMergedFromConfig)
}

func NewMergedFromConfig(ctx context.Context, other map[string]any) (api.Tariff, error) {
	cc := struct {
		Primary   Typed
		Secondary Typed
	}{}

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

	t := &Merged{
		log:       util.NewLogger("merged"),
		primary:   primary,
		secondary: secondary,
	}

	return t, nil
}

// firstIndexAt returns the index of the first rate starting at the given time, or -1 if not found.
func firstIndexAt(rates api.Rates, start time.Time) int {
	for i, r := range rates {
		if r.Start.Equal(start) {
			return i
		}
		if r.Start.After(start) {
			break
		}
	}
	return -1
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

	// Find where primary data ends
	primaryEnd := result[len(result)-1].End

	// Append secondary rates starting where primary ends
	if idx := firstIndexAt(secondaryRates, primaryEnd); idx >= 0 {
		return append(result, secondaryRates[idx:]...), nil
	}

	t.log.WARN.Printf("secondary tariff does not align gapless with primary, ignoring secondary")
	return result, nil
}

// Type implements the api.Tariff interface
func (t *Merged) Type() api.TariffType {
	return t.primary.Type()
}
