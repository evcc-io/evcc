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
	mu         sync.Mutex
	ctx        context.Context
	log        *util.Logger
	reset      []string
	initial    *string
	set        Config
	timeout    time.Duration
	transition bool // delay writes during mode transitions
	cancel     func()
}

func init() {
	registry.AddCtx("watchdog", NewWatchDogFromConfig)
}

// NewWatchDogFromConfig creates watchDog provider
func NewWatchDogFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Reset      []string
		Initial    *string
		Set        Config
		Timeout    time.Duration
		Transition bool
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &watchdogPlugin{
		ctx:        ctx,
		log:        util.ContextLoggerWithDefault(ctx, util.NewLogger("watchdog")),
		reset:      cc.Reset,
		initial:    cc.Initial,
		set:        cc.Set,
		timeout:    cc.Timeout,
		transition: cc.Transition,
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

type transitionState[T comparable] struct {
	currentMode     *T
	pendingMode     *T
	transitionTimer *time.Timer
	lastWrite       time.Time
}

// setter is the generic setter function for watchdogPlugin
// it is currently not possible to write this as a method
func setter[T comparable](o *watchdogPlugin, set func(T) error, reset []T) func(T) error {
	var state transitionState[T]

	return func(val T) error {
		o.mu.Lock()
		defer o.mu.Unlock()

		// Cancel pending transition
		if state.transitionTimer != nil {
			state.transitionTimer.Stop()
			state.transitionTimer = nil
			state.pendingMode = nil
		}

		// Calculate delay from last write
		requiredDelay := o.timeout + 5*time.Second
		timeSinceLastWrite := time.Since(state.lastWrite)
		actualDelay := max(0, requiredDelay-timeSinceLastWrite)

		// Check if delay needed: transition to non-reset mode with transition enabled or delay required
		if o.transition && state.currentMode != nil && !slices.Contains(reset, val) && actualDelay > 0 {

			// Stop running wdt
			if o.cancel != nil {
				o.cancel()
				o.cancel = nil
			}

			// Store pending mode
			valCopy := val
			state.pendingMode = &valCopy

			o.log.DEBUG.Printf("delayed transition scheduled: requiredDelay=%v, timeSinceLastWrite=%v, actualDelay=%v, from=%v, to=%v",
				requiredDelay, timeSinceLastWrite, actualDelay, state.currentMode, val)

			state.transitionTimer = time.AfterFunc(actualDelay, func() {
				o.mu.Lock()
				defer o.mu.Unlock()

				if state.pendingMode == nil {
					return
				}

				targetVal := *state.pendingMode
				state.pendingMode = nil
				state.transitionTimer = nil

				o.log.DEBUG.Printf("delayed transition executing: to=%v", targetVal)
				if err := set(targetVal); err != nil {
					o.log.ERROR.Printf("delayed transition failed: %v", err)
					return
				}

				state.lastWrite = time.Now()
				state.currentMode = &targetVal
				o.log.DEBUG.Printf("delayed transition completed: currentMode=%v", targetVal)

				if !slices.Contains(reset, targetVal) {
					var ctx context.Context
					ctx, o.cancel = context.WithCancel(context.Background())
					go o.wdt(ctx, func() error {
						o.mu.Lock()
						defer o.mu.Unlock()
						if err := set(targetVal); err != nil {
							return err
						}
						state.lastWrite = time.Now()
						return nil
					})
				}
			})

			return nil
		}

		// stop wdt on reset value
		if o.cancel != nil {
			o.cancel()
			o.cancel = nil
		}

		// start wdt on non-reset value
		if !slices.Contains(reset, val) {
			var ctx context.Context
			ctx, o.cancel = context.WithCancel(context.Background())
			go o.wdt(ctx, func() error {
				o.mu.Lock()
				defer o.mu.Unlock()
				if err := set(val); err != nil {
					return err
				}
				state.lastWrite = time.Now()
				return nil
			})
		}

		if err := set(val); err != nil {
			return err
		}

		state.lastWrite = time.Now()
		valCopy := val
		state.currentMode = &valCopy

		return nil
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
