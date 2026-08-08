package meter

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/tesla"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
)

type fleetConfig struct {
	tesla.FleetConfig `mapstructure:",squash"`
	SiteId            int64
	Other             map[string]any `mapstructure:",remain"`
}

func init() {
	registry.Add("powerwall-fleet", NewPowerWallFleetFromConfig)
}

// NewPowerWallFleetFromConfig creates a PowerWall meter with Fleet API battery control.
func NewPowerWallFleetFromConfig(other map[string]any) (api.Meter, error) {
	cc := fleetConfig{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	local, err := decodePowerWallConfig(cc.Other)
	if err != nil {
		return nil, err
	}

	if err := cc.FleetConfig.Validate(); err != nil {
		return nil, err
	}

	log := util.NewLogger("powerwall").Redact(
		local.User,
		local.Password,
		cc.Credentials.ID,
		cc.Credentials.Secret,
		cc.Tokens.Access,
		cc.Tokens.Refresh,
	)
	m, err := newPowerWall(log, local)
	if err != nil {
		return nil, err
	}

	energySite, err := teslaEnergySite(log, cc.FleetConfig, cc.SiteId)
	if err != nil {
		return nil, err
	}

	implement.May(m, implement.BatteryController(local.batterySocLimits.LimitController(func() (float64, error) {
		return socG(energySite)
	}, func(limit float64) error {
		return energySite.SetBatteryReserve(teslaReserveLimit(limit))
	})))

	return m, nil
}

func teslaReserveLimit(limit float64) uint64 {
	if math.IsNaN(limit) || limit <= 0 {
		return 0
	}
	if limit >= 100 {
		return 100
	}

	limitUint := uint64(limit)
	// Tesla firmware accepts values up to 80 or exactly 100 in this range.
	if limitUint > 80 {
		return 80
	}
	return limitUint
}

func teslaEnergySite(log *util.Logger, config tesla.FleetConfig, siteId int64) (*teslaclient.EnergySite, error) {
	fleet, err := config.Client(log)
	if err != nil {
		return nil, err
	}
	tc := fleet.Client

	if siteId == 0 {
		products, err := tc.Products()
		if err != nil {
			return nil, fmt.Errorf("discover energy sites: %w", err)
		}

		for _, p := range products {
			if p.EnergySiteId != 0 {
				siteId = p.EnergySiteId
				break
			}
		}

		if siteId == 0 {
			return nil, errors.New("no energy site found")
		}
	}

	log.Redact(strconv.FormatInt(siteId, 10))

	return tc.EnergySite(siteId)
}

func socG(energySite *teslaclient.EnergySite) (float64, error) {
	ess, err := energySite.EnergySiteStatus()
	if err != nil {
		return 0, fmt.Errorf("get energy site status: %w", err)
	}

	return math.Round(ess.PercentageCharged), nil
}
