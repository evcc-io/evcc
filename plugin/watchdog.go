package plugin

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
)

type watchdogPlugin struct {
	mu      sync.Mutex
	ctx     context.Context
	log     *util.Logger
	reset   *string
	set     Config
	timeout time.Duration
	cancel  func()
}

func init() {
	registry.AddCtx("watchdog", NewWatchDogFromConfig)
}

// NewWatchDogFromConfig creates watchDog provider
func NewWatchDogFromConfig(ctx context.Context, other map[string]interface{}) (Plugin, error) {
	var cc struct {
		Reset   *string
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
		set:     cc.Set,
		timeout: cc.Timeout,
	}

	return o, nil
}

func (o *watchdogPlugin) wdt(ctx context.Context, set func() error) {
	for tick := time.Tick(o.timeout); ; {
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
func setter[T comparable](o *watchdogPlugin, set func(T) error, reset *T) func(T) error {
	return func(val T) error {
		o.mu.Lock()

		// stop wdt on new write
		if o.cancel != nil {
			o.cancel()
			o.cancel = nil
		}

		// start wdt on non-reset value
		if reset == nil || val != *reset {
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

	var reset *int64
	if o.reset != nil {
		val, err := strconv.ParseInt(*o.reset, 10, 64)
		if err != nil {
			return nil, err
		}
		reset = &val
	}

	return setter(o, set, reset), nil
}

var _ FloatSetter = (*watchdogPlugin)(nil)

func (o *watchdogPlugin) FloatSetter(param string) (func(float64) error, error) {
	set, err := o.set.FloatSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	var reset *float64
	if o.reset != nil {
		val, err := strconv.ParseFloat(*o.reset, 64)
		if err != nil {
			return nil, err
		}
		reset = &val
	}

	return setter(o, set, reset), nil
}

var _ BoolSetter = (*watchdogPlugin)(nil)

func (o *watchdogPlugin) BoolSetter(param string) (func(bool) error, error) {
	set, err := o.set.BoolSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	var reset *bool
	if o.reset != nil {
		val, err := strconv.ParseBool(*o.reset)
		if err != nil {
			return nil, err
		}
		reset = &val
	}

	return setter(o, set, reset), nil
}
