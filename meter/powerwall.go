package meter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andig/go-powerwall"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// PowerWall is the tesla powerwall meter
type PowerWall struct {
	implement.Caps
	usage  string
	client *powerwall.Client
	meterG func() (map[string]powerwall.MeterAggregatesData, error)
}

type powerWallConfig struct {
	URI, Usage, User, Password string
	Cache                      time.Duration
	SiteId                     int64  // Fleet API only, deprecated for the local meter
	RefreshToken_              string `mapstructure:"refreshToken"` // TODO deprecated
	batteryCapacityCtx         `mapstructure:",squash"`
	batterySocLimitsCtx        `mapstructure:",squash"`
	batteryPowerLimitsCtx      `mapstructure:",squash"`
}

func defaultPowerWallConfig() powerWallConfig {
	return powerWallConfig{
		Cache: time.Second,
	}
}

// validate checks required parameters and maps legacy usage names
func (cc *powerWallConfig) validate() error {
	if cc.Usage == "" {
		return errors.New("missing usage")
	}

	if cc.Password == "" {
		return errors.New("missing password")
	}

	// support default meter names
	switch strings.ToLower(cc.Usage) {
	case "grid":
		cc.Usage = "site"
	case "pv":
		cc.Usage = "solar"
	}

	return nil
}

func init() {
	registry.AddCtx("tesla", NewPowerWallFromConfig)
	registry.AddCtx("powerwall", NewPowerWallFromConfig)
}

// NewPowerWallFromConfig creates a PowerWall Powerwall Meter from generic config
func NewPowerWallFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := defaultPowerWallConfig()
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.validate(); err != nil {
		return nil, err
	}

	log := util.NewLogger("powerwall").Redact(cc.User, cc.Password)

	if cc.RefreshToken_ != "" {
		log.WARN.Println("refreshToken is deprecated, use the Powerwall (Fleet API) template for battery control")
	}

	return newPowerWall(ctx, log, cc)
}

func newPowerWall(ctx context.Context, log *util.Logger, cc powerWallConfig) (*PowerWall, error) {
	httpClient := &http.Client{
		Transport: request.NewTripper(log, powerwall.DefaultTransport()),
		Timeout:   time.Second * 2, // Timeout after 2 seconds
	}

	client := powerwall.NewClient(cc.URI, cc.User, cc.Password, powerwall.WithHttpClient(httpClient))
	if _, err := client.GetStatus(); err != nil {
		return nil, err
	}

	m := &PowerWall{
		Caps:   implement.New(),
		client: client,
		usage:  strings.ToLower(cc.Usage),
		meterG: util.Cached(client.GetMetersAggregates, cc.Cache),
	}

	if m.usage == "load" || m.usage == "solar" {
		implement.Has(m, implement.MeterEnergy(m.totalEnergy))
	}

	if m.usage == "battery" {
		capacity, err := cc.batteryCapacityCtx.Decorator(ctx)
		if err != nil {
			return nil, err
		}

		minG, maxG, err := cc.batterySocLimitsCtx.getters(ctx)
		if err != nil {
			return nil, err
		}

		if minG == nil {
			opG := util.Cached(client.GetOperation, cc.Cache)
			minG = func() float64 {
				op, err := opG()
				if err != nil {
					return 0
				}
				return op.BackupReservePercent
			}
		}

		if maxG == nil {
			maxG = func() float64 { return 100 }
		}

		socLimiter := func() (float64, float64) {
			return minG(), maxG()
		}

		powerLimiter, err := cc.batteryPowerLimitsCtx.Decorator(ctx)
		if err != nil {
			return nil, err
		}

		if capacity == nil || powerLimiter == nil {
			statusG := util.Cached(client.GetSystemStatus, cc.Cache)

			// validate connectivity and gate power limiter capability
			res, err := statusG()
			if err != nil {
				return nil, err
			}

			if capacity == nil {
				capacity = func() float64 {
					res, err := statusG()
					if err != nil {
						return 0
					}
					return res.NominalFullPackEnergy / 1e3
				}
			}

			if powerLimiter == nil && res.MaxApparentPower > 0 {
				powerLimiter = func() (float64, float64) {
					res, err := statusG()
					if err != nil {
						return 0, 0
					}
					return res.MaxApparentPower, res.MaxApparentPower
				}
			}
		}

		implement.Has(m, implement.Battery(m.batterySoc))
		implement.May(m, implement.BatterySocLimiter(socLimiter))
		implement.May(m, implement.BatteryPowerLimiter(powerLimiter))
		implement.Has(m, implement.BatteryCapacity(capacity))
	}

	return m, nil
}

var _ api.Meter = (*PowerWall)(nil)

// CurrentPower implements the api.Meter interface
func (m *PowerWall) CurrentPower() (float64, error) {
	res, err := m.meterG()
	if err != nil {
		return 0, err
	}

	if o, ok := res[m.usage]; ok {
		return float64(o.InstantPower), nil
	}

	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// totalEnergy implements the api.MeterEnergy interface
func (m *PowerWall) totalEnergy() (float64, error) {
	res, err := m.meterG()
	if err != nil {
		return 0, err
	}

	if o, ok := res[m.usage]; ok {
		switch m.usage {
		case "load":
			return float64(o.EnergyImported) / 1e3, nil
		case "solar":
			return float64(o.EnergyExported) / 1e3, nil
		}
	}

	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// batterySoc implements the api.Battery interface
func (m *PowerWall) batterySoc() (float64, error) {
	res, err := m.client.GetSOE()
	if err != nil {
		return 0, err
	}

	return res.Percentage, err
}
