package measurement

import (
	"context"
	"errors"
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

	if (dimS == nil) != (dimmedG == nil) {
		return nil, nil, errors.New("must have dim and dimmed or none of both")
	}

	return dimS, dimmedG, nil
}

type Curtailer struct {
	Curtail   *plugin.Config // optional
	Curtailed *plugin.Config // optional
}

func (cc *Curtailer) Configure(ctx context.Context) (
	func(bool) error,
	func() (bool, error),
	error,
) {
	curtailS, err := cc.Curtail.BoolSetter(ctx, "curtail")
	if err != nil {
		return nil, nil, fmt.Errorf("curtail: %w", err)
	}

	curtailedG, err := cc.Curtailed.BoolGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("curtailed: %w", err)
	}

	if (curtailS == nil) != (curtailedG == nil) {
		return nil, nil, errors.New("must have curtail and curtailed or none of both")
	}

	return curtailS, curtailedG, nil
}
