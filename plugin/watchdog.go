package plugin

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
)

type watchdogPlugin struct {
	mu       sync.Mutex
	ctx      context.Context
	log      *util.Logger
	reset    []string
	initial  *string
	set      Config
	timeout  time.Duration
	deferred bool
	cancel   func()
	clock    clock.Clock
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
		Defer   bool `mapstructure:"defer"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &watchdogPlugin{
		ctx:      ctx,
		log:      util.ContextLoggerWithDefault(ctx, util.NewLogger("watchdog")),
		reset:    cc.Reset,
		initial:  cc.Initial,
		set:      cc.Set,
		timeout:  cc.Timeout,
		deferred: cc.Defer,
		clock:    clock.New(),
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

type deferredState[T comparable] struct {
	val   T
	timer *clock.Timer
}

// setter is the generic setter function for watchdogPlugin
// it is currently not possible to write this as a method
func setter[T comparable](o *watchdogPlugin, set func(T) error, reset []T) func(T) error {
	var state *deferredState[T]
	var lastUpdated time.Time
	var last *T

	// stop running wdt
	stopWdt := func() {
		if o.cancel != nil {
			o.cancel()
			o.cancel = nil
		}
	}

	// set value and start wdt
	setAndStartWdt := func(val T) error {
		if err := set(val); err != nil {
			return err
		}
		lastUpdated = o.clock.Now()
		last = &val

		// start wdt for non-reset value
		if !slices.Contains(reset, val) {
			var ctx context.Context
			ctx, o.cancel = context.WithCancel(context.Background())

			go o.wdt(ctx, func() error {
				o.mu.Lock()
				defer o.mu.Unlock()

				if err := set(val); err != nil {
					return err
				}
				lastUpdated = o.clock.Now()

				return nil
			})
		}

		return nil
	}

	return func(val T) error {
		o.mu.Lock()
		defer o.mu.Unlock()

		// cancel deferred update
		if state != nil {
			state.timer.Stop()
			state = nil
		}

		// if value unchanged, let wdt continue running
		// TODO refactor use of last value once batterymode is set only once, currently required to avoid defer loops
		if last != nil && *last == val && o.cancel != nil {
			return nil
		}

		// calculate remaining deferred delay
		delay := max(0, o.timeout+5*time.Second-o.clock.Since(lastUpdated))

		// defer update to non-reset value
		if o.deferred && delay > 0 && !lastUpdated.IsZero() && !slices.Contains(reset, val) {
			stopWdt()

			// store deferred value
			state = &deferredState[T]{
				val: val,
				timer: o.clock.AfterFunc(delay, func() {
					o.mu.Lock()
					defer o.mu.Unlock()

					state = nil

					o.log.TRACE.Printf("deferred update executing: to=%v", val)
					if err := setAndStartWdt(val); err != nil {
						o.log.ERROR.Printf("deferred update failed: %v", err)
						return
					}
					o.log.TRACE.Printf("deferred update completed: value=%v", val)
				}),
			}

			return nil
		}

		// immediate update
		stopWdt()
		return setAndStartWdt(val)
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
