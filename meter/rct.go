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
	conn  *rct.Connection // connection with the RCT device
	usage string          // grid, pv, battery
}

var (
	rctMu    sync.Mutex
	rctCache = make(map[string]*rct.Connection)
)

func init() {
	registry.AddCtx("rct", NewRCTFromConfig)
}

//go:generate go tool decorate -f decorateRCT -b *RCT -r api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error" -t "api.BatteryCapacity,Capacity,func() float64"

// NewRCTFromConfig creates an RCT from generic config
func NewRCTFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		capacity       `mapstructure:",squash"`
		Uri, Usage     string
		MinSoc, MaxSoc int
		Cache          time.Duration
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	return NewRCT(ctx, cc.Uri, cc.Usage, cc.MinSoc, cc.MaxSoc, cc.Cache, cc.capacity.Decorator())
}

// NewRCT creates an RCT meter
func NewRCT(ctx context.Context, uri, usage string, minSoc, maxSoc int, cache time.Duration, capacity func() float64) (api.Meter, error) {
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
		usage: strings.ToLower(usage),
		conn:  conn,
	}

	// decorate api.MeterEnergy
	var totalEnergy func() (float64, error)
	if usage == "grid" {
		totalEnergy = m.totalEnergy
	}

	// decorate api.BatterySoc
	var batterySoc func() (float64, error)
	var batteryMode func(api.BatteryMode) error
	if usage == "battery" {
		batterySoc = m.batterySoc

		batteryMode = func(mode api.BatteryMode) error {
			if mode != api.BatteryNormal {
				batStatus, err := m.queryInt32(rct.BatteryBatStatus)
				if err != nil {
					return err
				}

				// see https://github.com/weltenwort/home-assistant-rct-power-integration/issues/264#issuecomment-2124811644
				if batStatus != 0 {
					return errors.New("invalid battery operating mode")
				}
			}

			switch mode {
			case api.BatteryNormal:
				if err := m.conn.Write(rct.PowerMngSocStrategy, []byte{rct.SOCTargetInternal}); err != nil {
					return err
				}

				if err := m.conn.Write(rct.BatterySoCTargetMin, m.floatVal(float32(minSoc)/100)); err != nil {
					return err
				}

				return m.conn.Write(rct.PowerMngBatteryPowerExternW, m.floatVal(float32(0)))

			case api.BatteryHold:
				if err := m.conn.Write(rct.PowerMngSocStrategy, []byte{rct.SOCTargetInternal}); err != nil {
					return err
				}

				return m.conn.Write(rct.BatterySoCTargetMin, m.floatVal(float32(maxSoc)/100))

			case api.BatteryCharge:
				if err := m.conn.Write(rct.PowerMngUseGridPowerEnable, []byte{1}); err != nil {
					return err
				}

				if err := m.conn.Write(rct.PowerMngBatteryPowerExternW, m.floatVal(float32(-10_000))); err != nil {
					return err
				}

				return m.conn.Write(rct.PowerMngSocStrategy, []byte{rct.SOCTargetExternal})

			default:
				return api.ErrNotAvailable
			}
		}
	}

	return decorateRCT(m, totalEnergy, batterySoc, batteryMode, capacity), nil
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

		eg.Go(func() error {
			var err error
			c, err = m.queryFloat(rct.S0ExternalPowerW)
			return err
		})

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

// batterySoc implements the api.Battery interface
func (m *RCT) batterySoc() (float64, error) {
	res, err := m.queryFloat(rct.BatterySoC)
	return res * 100, err
}

func (m *RCT) bo() *backoff.ExponentialBackOff {
	return backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(500*time.Millisecond),
		backoff.WithMaxInterval(2*time.Second),
		backoff.WithMaxElapsedTime(10*time.Second))
}

// queryFloat adds retry logic of recoverable errors to QueryFloat32
func (m *RCT) queryFloat(id rct.Identifier) (float64, error) {
	res, err := backoff.RetryWithData(func() (float32, error) {
		return m.conn.QueryFloat32(id)
	}, m.bo())
	return float64(res), err
}

// queryInt32 adds retry logic of recoverable errors to QueryInt32
func (m *RCT) queryInt32(id rct.Identifier) (int32, error) {
	res, err := backoff.RetryWithData(func() (int32, error) {
		return m.conn.QueryInt32(id)
	}, m.bo())
	return res, err
}
