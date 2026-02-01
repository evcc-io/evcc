package tariff

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Extended combines a primary tariff with a secondary (forecast) tariff.
// Primary rates are used where available, secondary fills gaps after primary ends.
type Extended struct {
	log       *util.Logger
	primary   api.Tariff
	secondary api.Tariff
}

var _ api.Tariff = (*Extended)(nil)

func init() {
	registry.AddCtx("extended", NewExtendedFromConfig)
}

func NewExtendedFromConfig(ctx context.Context, other map[string]any) (api.Tariff, error) {
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

	pType, sType := primary.Type(), secondary.Type()
	if pType != sType {
		return nil, errors.New("primary and secondary tariff types are not compatible")
	}

	t := &Extended{
		log:       util.NewLogger("extended"),
		primary:   primary,
		secondary: secondary,
	}

	return t, nil
}

// Rates implements the api.Tariff interface
func (t *Extended) Rates() (api.Rates, error) {
	primaryRates, err := t.primary.Rates()
	if err != nil {
		t.log.DEBUG.Printf("primary tariff failed, falling back to secondary: %v", err)
		return t.secondary.Rates()
	}

	secondaryRates, err := t.secondary.Rates()
	if err != nil {
		t.log.DEBUG.Printf("secondary tariff failed, using primary only: %v", err)
		return primaryRates, nil
	}

	// Find where primary data ends
	var primaryEnd time.Time
	if len(primaryRates) > 0 {
		primaryRates.Sort()
		primaryEnd = primaryRates[len(primaryRates)-1].End
	}

	// Add secondary rates that start at or after primary ends
	res := slices.Clone(primaryRates)
	for _, r := range secondaryRates {
		if !r.Start.Before(primaryEnd) {
			res = append(res, r)
		}
	}
	res.Sort()

	return res, nil
}

// Type implements the api.Tariff interface
func (t *Extended) Type() api.TariffType {
	return t.primary.Type()
}
