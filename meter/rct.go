package meter

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/rct"
	"golang.org/x/sync/errgroup"
)

// RCT implements the api.Meter interface
type RCT struct {
	conn          *rct.Connection // connection with the RCT device
	usage         string          // grid, pv, battery
	externalPower bool            // whether to query external power
}

var (
	rctMu    sync.Mutex
	rctCache = make(map[string]*rct.Connection)
)

func init() {
	registry.AddCtx("rct", NewRCTFromConfig)
}

//go:generate go tool decorate -f decorateRCT -b *RCT -r api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.BatterySocLimiter,GetSocLimits,func() (float64, float64)" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error" -t "api.BatteryCapacity,Capacity,func() float64"

// NewRCTFromConfig creates an RCT from generic config
func NewRCTFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		batterySocLimits `mapstructure:",squash"`
		Uri, Usage       string
		MaxChargePower   int
		Capacity         float64
		Capacity2        float64
		ExternalPower    bool
		Cache            time.Duration
	}{
		batterySocLimits: batterySocLimits{
			MinSoc: 20,
			MaxSoc: 95,
		},
		MaxChargePower: 10000,
		Cache:          30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	return NewRCT(ctx, cc.Uri, cc.Usage, cc.batterySocLimits, cc.MaxChargePower, cc.Cache, cc.ExternalPower, cc.Capacity, cc.Capacity2)
}

// NewRCT creates an RCT meter
func NewRCT(ctx context.Context, uri, usage string, batterySocLimits batterySocLimits, maxchargepower int, cache time.Duration, externalPower bool, capacity float64, capacity2 float64) (api.Meter, error) {
	log := util.NewLogger("rct")

	// re-use connections
	rctMu.Lock()
	conn, ok := rctCache[uri]
	if !ok {
		var err error
		conn, err = rct.NewConnection(ctx, uri, rct.WithErrorCallback(func(err error) {
			if err != nil {
				log.ERROR.Println(err)
			}
		}), rct.WithLogger(log.TRACE.Printf), rct.WithTimeout(cache))
		if err != nil {
			rctMu.Unlock()
			return nil, err
		}

		rctCache[uri] = conn
	}
	rctMu.Unlock()

	m := &RCT{
		usage:         strings.ToLower(usage),
		conn:          conn,
		externalPower: externalPower,
	}

	// decorate api.MeterEnergy
	var totalEnergy func() (float64, error)
	if usage == "grid" {
		totalEnergy = m.totalEnergy
	}

	// decorate api.Battery
	var batterySoc func() (float64, error)
	var batterySocLimiter func() (float64, float64)
	var batteryCapacity func() float64
	var batteryMode func(api.BatteryMode) error

	if usage == "battery" {
		// validate capacity configuration for dual battery setups
		if capacity2 > 0 && capacity == 0 {
			return nil, errors.New("missing first battery capacity")
		}

		batterySoc = func() (float64, error) {
			soc, err := m.queryFloat(rct.BatterySoC)
			if err != nil {
				return 0, err
			}

			if capacity2 == 0 {
				return soc * 100, err
			}

			soc2, err := m.queryFloat(rct.BatteryPlaceholder0Soc)
			return (soc*capacity + soc2*capacity2) / (capacity + capacity2) * 100, err
		}

		batterySocLimiter = batterySocLimits.Decorator()

		if capacity != 0 {
			batteryCapacity = func() float64 { return capacity + capacity2 }
		}

		batteryMode = func(mode api.BatteryMode) error {
			if mode != api.BatteryNormal {
				batStatus, err := m.queryInt32(rct.BatteryStatus2)
				if err != nil {
					return err
				}

				// see https://github.com/weltenwort/home-assistant-rct-power-integration/issues/264#issuecomment-2124811644
				if batStatus != 0 {
					return errors.New("invalid battery operating mode")
				}
			}

			var eg errgroup.Group

			switch mode {
			case api.BatteryNormal:
				eg.Go(func() error {
					return m.conn.Write(rct.PowerMngSocStrategy, []byte{rct.SOCTargetInternal})
				})

				eg.Go(func() error {
					return m.conn.Write(rct.BatterySoCTargetMin, m.floatVal(float32(batterySocLimits.MinSoc)/100))
				})

				eg.Go(func() error {
					return m.conn.Write(rct.PowerMngBatteryPowerExternW, m.floatVal(float32(0)))
				})

			case api.BatteryHold:
				eg.Go(func() error {
					return m.conn.Write(rct.PowerMngSocStrategy, []byte{rct.SOCTargetInternal})
				})

				eg.Go(func() error {
					return m.conn.Write(rct.BatterySoCTargetMin, m.floatVal(float32(batterySocLimits.MaxSoc)/100))
				})

			case api.BatteryCharge:
				eg.Go(func() error {
					return m.conn.Write(rct.PowerMngUseGridPowerEnable, []byte{1})
				})

				eg.Go(func() error {
					return m.conn.Write(rct.PowerMngBatteryPowerExternW, m.floatVal(float32(-maxchargepower)))
				})

				eg.Go(func() error {
					return m.conn.Write(rct.PowerMngSocStrategy, []byte{rct.SOCTargetExternal})
				})

			default:
				return api.ErrNotAvailable
			}

			return eg.Wait()
		}
	}

	return decorateRCT(m, totalEnergy, batterySoc, batterySocLimiter, batteryMode, batteryCapacity), nil
}

func (m *RCT) floatVal(f float32) []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, math.Float32bits(f))
	return data
}

// CurrentPower implements the api.Meter interface
func (m *RCT) CurrentPower() (float64, error) {
	switch m.usage {
	case "grid":
		return m.queryFloat(rct.TotalGridPowerW)

	case "pv":
		var eg errgroup.Group
		var a, b, c float64

		eg.Go(func() error {
			var err error
			a, err = m.queryFloat(rct.SolarGenAPowerW)
			return err
		})

		eg.Go(func() error {
			var err error
			b, err = m.queryFloat(rct.SolarGenBPowerW)
			return err
		})

		if m.externalPower {
			eg.Go(func() error {
				var err error
				c, err = m.queryFloat(rct.S0ExternalPowerW)
				return err
			})
		}

		err := eg.Wait()
		return a + b + c, err

	case "battery":
		return m.queryFloat(rct.BatteryPowerW)

	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

// totalEnergy implements the api.MeterEnergy interface
func (m *RCT) totalEnergy() (float64, error) {
	switch m.usage {
	case "grid":
		res, err := m.queryFloat(rct.TotalEnergyGridWh)
		return res / 1000, err

	case "pv":
		var eg errgroup.Group
		var a, b float64

		eg.Go(func() error {
			var err error
			a, err = m.queryFloat(rct.TotalEnergySolarGenAWh)
			return err
		})

		eg.Go(func() error {
			var err error
			b, err = m.queryFloat(rct.TotalEnergySolarGenBWh)
			return err
		})

		err := eg.Wait()
		return (a + b) / 1000, err

	case "battery":
		var eg errgroup.Group
		var in, out float64

		eg.Go(func() error {
			var err error
			in, err = m.queryFloat(rct.TotalEnergyBattInWh)
			return err
		})

		eg.Go(func() error {
			var err error
			out, err = m.queryFloat(rct.TotalEnergyBattOutWh)
			return err
		})

		err := eg.Wait()
		return (in - out) / 1000, err

	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

func queryRCT[T any](id rct.Identifier, fun func(id rct.Identifier) (T, error)) (T, error) {
	bo := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(500*time.Millisecond),
		backoff.WithMaxInterval(2*time.Second),
		backoff.WithMaxElapsedTime(10*time.Second))

	return backoff.RetryWithData(func() (T, error) {
		return fun(id)
	}, bo)
}

// queryFloat adds retry logic of recoverable errors to QueryFloat32
func (m *RCT) queryFloat(id rct.Identifier) (float64, error) {
	res, err := queryRCT(id, m.conn.QueryFloat32)
	return float64(res), err
}

// queryInt32 adds retry logic of recoverable errors to QueryInt32
func (m *RCT) queryInt32(id rct.Identifier) (int32, error) {
	return queryRCT(id, m.conn.QueryInt32)
}
