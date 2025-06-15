package plugin

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
)

type watchdogPlugin struct {
	mu      sync.Mutex
	ctx     context.Context
	log     *util.Logger
	reset   []string
	initial *string
	set     Config
	timeout time.Duration
	cancel  func()
}

func init() {
	registry.AddCtx("watchdog", NewWatchDogFromConfig)
}

// NewWatchDogFromConfig creates watchDog provider
func NewWatchDogFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Reset   []string
		Initial *string
		Set     Config
		Timeout time.Duration
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &watchdogPlugin{
		ctx:     ctx,
		log:     contextLogger(ctx, util.NewLogger("watchdog")),
		reset:   cc.Reset,
		initial: cc.Initial,
		set:     cc.Set,
		timeout: cc.Timeout,
	}

	return o, nil
}

func (o *watchdogPlugin) wdt(ctx context.Context, set func() error) {
	for tick := time.Tick(o.timeout / 2); ; {
		select {
		case <-tick:
			if err := set(); err != nil {
				o.log.ERROR.Println(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// setter is the generic setter function for watchdogPlugin
// it is currently not possible to write this as a method
func setter[T comparable](o *watchdogPlugin, set func(T) error, reset []T) func(T) error {
	return func(val T) error {
		o.mu.Lock()

		// stop wdt on new write
		if o.cancel != nil {
			o.cancel()
			o.cancel = nil
		}

		// start wdt on non-reset value
		if !slices.Contains(reset, val) {
			var ctx context.Context
			ctx, o.cancel = context.WithCancel(context.Background())

			go o.wdt(ctx, func() error {
				return set(val)
			})
		}

		o.mu.Unlock()

		return set(val)
	}
}

var _ IntSetter = (*watchdogPlugin)(nil)

func (o *watchdogPlugin) IntSetter(param string) (func(int64) error, error) {
	set, err := o.set.IntSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	var reset []int64
	if o.reset != nil {
		for _, v := range o.reset {
			val, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
			reset = append(reset, val)
		}
	}

	res := setter(o, set, reset)
	if o.initial != nil {
		val, err := strconv.ParseInt(*o.initial, 10, 64)
		if err != nil {
			return nil, err
		}

		if err := res(val); err != nil {
			return nil, err
		}
	}

	return res, nil
}

var _ FloatSetter = (*watchdogPlugin)(nil)

func (o *watchdogPlugin) FloatSetter(param string) (func(float64) error, error) {
	set, err := o.set.FloatSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	var reset []float64
	if o.reset != nil {
		for _, v := range o.reset {
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, err
			}
			reset = append(reset, val)
		}
	}

	res := setter(o, set, reset)
	if o.initial != nil {
		val, err := strconv.ParseFloat(*o.initial, 64)
		if err != nil {
			return nil, err
		}

		if err := res(val); err != nil {
			return nil, err
		}
	}

	return res, nil
}

var _ BoolSetter = (*watchdogPlugin)(nil)

func (o *watchdogPlugin) BoolSetter(param string) (func(bool) error, error) {
	set, err := o.set.BoolSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	var reset []bool
	if len(o.reset) > 1 {
		return nil, fmt.Errorf("more than one boolean reset value")
	} else if len(o.reset) == 1 {
		val, err := strconv.ParseBool(o.reset[0])
		if err != nil {
			return nil, err
		}
		reset = append(reset, val)
	}

	res := setter(o, set, reset)
	if o.initial != nil {
		val, err := strconv.ParseBool(*o.initial)
		if err != nil {
			return nil, err
		}

		if err := res(val); err != nil {
			return nil, err
		}
	}

	return res, nil
}
