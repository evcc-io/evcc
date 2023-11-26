package provider

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
)

type watchdogProvider struct {
	mu      sync.Mutex
	log     *util.Logger
	reset   *string
	set     Config
	timeout time.Duration
	cancel  func()
}

func init() {
	registry.Add("watchdog", NewWatchDogFromConfig)
}

// NewWatchDogFromConfig creates watchDog provider
func NewWatchDogFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Reset   *string
		Set     Config
		Timeout time.Duration
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &watchdogProvider{
		log:     util.NewLogger("watchdog"),
		reset:   cc.Reset,
		set:     cc.Set,
		timeout: cc.Timeout,
	}

	return o, nil
}

func (o *watchdogProvider) wdt(ctx context.Context, set func() error) {
	tick := time.NewTicker(o.timeout / 2)
	for range tick.C {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		default:
			if err := set(); err != nil {
				o.log.ERROR.Println(err)
			}
		}
	}
}

var _ SetIntProvider = (*watchdogProvider)(nil)

func (o *watchdogProvider) IntSetter(param string) (func(int64) error, error) {
	set, err := NewIntSetterFromConfig(param, o.set)
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

	return func(val int64) error {
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
	}, err
}

// var _ SetFloatProvider = (*watchdogProvider)(nil)

// func (o *watchdogProvider) FloatSetter(param string) (func(float64) error, error) {
// 	set, err := NewFloatSetterFromConfig(param, o.set)
// 	if err != nil {
// 		return nil, err
// 	}

// 	val, err := strconv.ParseFloat(o.str, 64)
// 	return func(_ float64) error {
// 		return set(val)
// 	}, err
// }

// var _ SetBoolProvider = (*watchdogProvider)(nil)

// func (o *watchdogProvider) BoolSetter(param string) (func(bool) error, error) {
// 	set, err := NewBoolSetterFromConfig(param, o.set)
// 	if err != nil {
// 		return nil, err
// 	}

// 	val, err := strconv.ParseBool(o.str)
// 	return func(_ bool) error {
// 		return set(val)
// 	}, err
// }
