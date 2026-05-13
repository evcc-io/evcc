package measurement

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/plugin"
)

type Dimmer struct {
	Dim    *plugin.Config // optional
	Dimmed *plugin.Config // optional
}

func (cc *Dimmer) Configure(ctx context.Context) (
	func(bool) error,
	func() (bool, error),
	error,
) {
	dimS, err := cc.Dim.BoolSetter(ctx, "dim")
	if err != nil {
		return nil, nil, fmt.Errorf("dim: %w", err)
	}

	dimmedG, err := cc.Dimmed.BoolGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("dimmed: %w", err)
	}

	return dimS, dimmedG, nil
}
