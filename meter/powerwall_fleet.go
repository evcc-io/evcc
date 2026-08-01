package meter

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/tesla"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
	"golang.org/x/oauth2"
)

type fleetConfig struct {
	Credentials struct{ ID, Secret string }
	Tokens      struct{ Access, Refresh string }
	SiteId      int64
	Other       map[string]any `mapstructure:",remain"`
}

func init() {
	registry.Add("powerwall-fleet", NewPowerWallFleetFromConfig)
}

// NewPowerWallFleetFromConfig creates a PowerWall meter with Fleet API battery control.
func NewPowerWallFleetFromConfig(other map[string]any) (api.Meter, error) {
	localOther := make(map[string]any, len(other))
	for key, value := range other {
		if key != "credentials" && key != "tokens" && key != "siteId" {
			localOther[key] = value
		}
	}

	local, err := decodePowerWallConfig(localOther)
	if err != nil {
		return nil, err
	}

	cc := fleetConfig{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Credentials.ID == "" {
		return nil, errors.New("missing client id")
	}

	if cc.Tokens.Access == "" || cc.Tokens.Refresh == "" {
		return nil, api.ErrMissingToken
	}

	m, err := newPowerWall(local)
	if err != nil {
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
	energySite, err := teslaEnergySite(log, cc.Credentials.ID, cc.Credentials.Secret, cc.Tokens.Access, cc.Tokens.Refresh, cc.SiteId)
	if err != nil {
		return nil, err
	}

	implement.May(m, implement.BatteryController(local.batterySocLimits.LimitController(func() (float64, error) {
		return socG(energySite)
	}, func(limit float64) error {
		limitUint := uint64(limit)
		// Tesla firmware accepts values up to 80 or exactly 100 in this range.
		if limitUint > 80 && limitUint < 100 {
			limitUint = 80
		}
		return energySite.SetBatteryReserve(limitUint)
	})))

	return m, nil
}

func teslaEnergySite(log *util.Logger, clientID, clientSecret, accessToken, refreshToken string, siteId int64) (*teslaclient.EnergySite, error) {
	identity, err := tesla.NewIdentity(log, tesla.OAuth2Config(clientID, clientSecret), &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       time.Now(),
	})
	if err != nil {
		return nil, err
	}

	hc := request.NewClient(log)
	hc.Transport = &oauth2.Transport{
		Source: identity,
		Base:   hc.Transport,
	}

	tc, err := teslaclient.NewClient(context.Background(), teslaclient.WithClient(hc))
	if err != nil {
		return nil, err
	}

	region, err := tc.UserRegion()
	if err != nil {
		return nil, err
	}
	tc.SetBaseUrl(region.FleetApiBaseUrl)

	if siteId == 0 {
		products, err := tc.Products()
		if err != nil {
			return nil, err
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
		return 0, err
	}

	return math.Round(ess.PercentageCharged), nil
}
