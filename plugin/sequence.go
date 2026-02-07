package plugin

import (
	"context"

	"github.com/evcc-io/evcc/util"
)

type sequencePlugin struct {
	ctx context.Context
	set []Config
}

func init() {
	registry.AddCtx("sequence", NewSequenceFromConfig)
}

// NewSequenceFromConfig creates sequence provider
func NewSequenceFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Set []Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &sequencePlugin{
		ctx: ctx,
		set: cc.Set,
	}

	return o, nil
}

var _ IntSetter = (*sequencePlugin)(nil)

func (o *sequencePlugin) IntSetter(param string) (func(int64) error, error) {
	set := make([]func(int64) error, 0, len(o.set))
	for _, cc := range o.set {
		s, err := cc.IntSetter(o.ctx, param)
		if err != nil {
			return nil, err
		}
		set = append(set, s)
	}

	return func(val int64) error {
		for _, s := range set {
			if err := s(val); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

var _ FloatSetter = (*sequencePlugin)(nil)

func (o *sequencePlugin) FloatSetter(param string) (func(float64) error, error) {
	set := make([]func(float64) error, 0, len(o.set))
	for _, cc := range o.set {
		s, err := cc.FloatSetter(o.ctx, param)
		if err != nil {
			return nil, err
		}
		set = append(set, s)
	}

	return func(val float64) error {
		for _, s := range set {
			if err := s(val); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

var _ BoolSetter = (*sequencePlugin)(nil)

func (o *sequencePlugin) BoolSetter(param string) (func(bool) error, error) {
	set := make([]func(bool) error, 0, len(o.set))
	for _, cc := range o.set {
		s, err := cc.BoolSetter(o.ctx, param)
		if err != nil {
			return nil, err
		}
		set = append(set, s)
	}

	return func(val bool) error {
		for _, s := range set {
			if err := s(val); err != nil {
				return err
			}
		}
		return nil
	}, nil
}
