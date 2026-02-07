package plugin

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/util"
)

type sleepPlugin struct {
	duration time.Duration
}

func init() {
	registry.AddCtx("sleep", NewSleepFromConfig)
}

// NewSleepFromConfig creates sleep provider
func NewSleepFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Duration time.Duration
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &sleepPlugin{
		duration: cc.Duration,
	}

	return o, nil
}

// sleeper is the generic sleeper function for sleepPlugin
// it is currently not possible to write this as a method
func sleeper[T comparable](o *sleepPlugin) func(T) error {
	return func(val T) error {
		<-time.After(o.duration)

		return nil
	}
}

var _ IntSetter = (*sleepPlugin)(nil)

func (o *sleepPlugin) IntSetter(param string) (func(int64) error, error) {
	return sleeper[int64](o), nil
}

var _ FloatSetter = (*sleepPlugin)(nil)

func (o *sleepPlugin) FloatSetter(param string) (func(float64) error, error) {
	return sleeper[float64](o), nil
}

var _ BoolSetter = (*sleepPlugin)(nil)

func (o *sleepPlugin) BoolSetter(param string) (func(bool) error, error) {
	return sleeper[bool](o), nil
}
