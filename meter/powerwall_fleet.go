package meter

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/tesla"
)

type fleetConfig struct {
	powerWallConfig   `mapstructure:",squash"`
	tesla.FleetConfig `mapstructure:",squash"`
}

func init() {
	registry.Add("powerwall-fleet", NewPowerWallFleetFromConfig)
}

// NewPowerWallFleetFromConfig creates a PowerWall meter with Fleet API battery control
func NewPowerWallFleetFromConfig(other map[string]any) (api.Meter, error) {
	cc := fleetConfig{powerWallConfig: defaultPowerWallConfig()}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.validate(); err != nil {
		return nil, err
	}

	if err := cc.FleetConfig.Validate(); err != nil {
		return nil, err
	}

	log := util.NewLogger("powerwall").Redact(
		cc.User,
		cc.Password,
		cc.Credentials.ID,
		cc.Credentials.Secret,
		cc.Tokens.Access,
		cc.Tokens.Refresh,
	)
	m, err := newPowerWall(log, cc.powerWallConfig)
	if err != nil {
		return nil, err
	}

	fleet, siteId, err := teslaEnergySite(log, cc.FleetConfig, cc.SiteId)
	if err != nil {
		return nil, err
	}

	// charge mode raises the reserve to full, normal mode restores the configured minimum
	limits := batterySocLimits{MinSoc: cc.MinSoc, MaxSoc: 100}

	implement.Has(m, implement.BatteryController(batteryModesSocLimit, limits.LimitController(func() (float64, error) {
		ess, err := fleet.EnergySiteStatus(siteId)
		if err != nil {
			return 0, fmt.Errorf("get energy site status: %w", err)
		}

		return math.Round(ess.PercentageCharged), nil
	}, func(limit float64) error {
		return fleet.SetBatteryReserve(siteId, teslaReserveLimit(limit))
	})))

	return m, nil
}

// teslaReserveLimit clamps the reserve to values accepted by Tesla firmware: up to 80 or exactly 100
func teslaReserveLimit(limit float64) uint64 {
	switch {
	case limit >= 100:
		return 100
	case limit > 80:
		return 80
	case limit > 0:
		return uint64(limit)
	default: // negative or NaN
		return 0
	}
}

func teslaEnergySite(log *util.Logger, config tesla.FleetConfig, siteId int64) (*tesla.API, int64, error) {
	fleet, err := config.Client(log)
	if err != nil {
		return nil, 0, err
	}

	if siteId == 0 {
		products, err := fleet.Products()
		if err != nil {
			return nil, 0, fmt.Errorf("discover energy sites: %w", err)
		}

		for _, p := range products {
			if p.EnergySiteId != 0 {
				siteId = p.EnergySiteId
				break
			}
		}

		if siteId == 0 {
			return nil, 0, errors.New("no energy site found")
		}
	}

	log.Redact(strconv.FormatInt(siteId, 10))

	return fleet, siteId, nil
}
