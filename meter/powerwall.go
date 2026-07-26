package meter

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andig/go-powerwall"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/tesla"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
	"golang.org/x/oauth2"
)

// PowerWall is the tesla powerwall meter
type PowerWall struct {
	implement.Caps
	usage      string
	client     *powerwall.Client
	meterG     func() (map[string]powerwall.MeterAggregatesData, error)
	energySite *teslaclient.EnergySite
}

func init() {
	registry.Add("tesla", NewPowerWallFromConfig)
	registry.Add("powerwall", NewPowerWallFromConfig)
}

// NewPowerWallFromConfig creates a PowerWall Powerwall Meter from generic config
func NewPowerWallFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI, Usage, User, Password string
		Cache                      time.Duration
		Credentials                struct{ ID, Secret string }
		Tokens                     struct{ Access, Refresh string }
		RefreshToken               string // deprecated
		SiteId                     int64
		batterySocLimits           `mapstructure:",squash"`
		batteryPowerLimits         `mapstructure:",squash"`
	}{
		batterySocLimits: batterySocLimits{
			MinSoc: 20,
			MaxSoc: 95,
		},
		batteryPowerLimits: batteryPowerLimits{
			MaxChargePower:    4600,
			MaxDischargePower: 4600,
		},
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	if cc.Password == "" {
		return nil, errors.New("missing password")
	}

	// the owner api used by the refresh token has been retired by tesla
	if cc.RefreshToken != "" {
		return nil, errors.New("battery control requires tesla fleet api credentials, see https://docs.evcc.io/docs/devices/meters#tesla-powerwall")
	}

	// support default meter names
	switch strings.ToLower(cc.Usage) {
	case "grid":
		cc.Usage = "site"
	case "pv":
		cc.Usage = "solar"
	}

	log := util.NewLogger("powerwall").Redact(cc.User, cc.Password,
		cc.Credentials.ID, cc.Credentials.Secret, cc.Tokens.Access, cc.Tokens.Refresh)

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

	batteryControl := cc.Credentials.ID != "" || cc.Tokens.Access != "" || cc.Tokens.Refresh != ""

	if batteryControl {
		if cc.Credentials.ID == "" {
			return nil, errors.New("missing client id")
		}

		if cc.Tokens.Access == "" {
			return nil, api.ErrMissingToken
		}

		energySite, err := teslaEnergySite(log, cc.Credentials.ID, cc.Credentials.Secret, cc.Tokens.Access, cc.Tokens.Refresh, cc.SiteId)
		if err != nil {
			return nil, err
		}
		m.energySite = energySite
	}

	if m.usage == "load" || m.usage == "solar" {
		implement.Has(m, implement.MeterEnergy(m.totalEnergy))
	}

	if m.usage == "battery" {
		implement.Has(m, implement.Battery(m.batterySoc))
		implement.May(m, implement.BatterySocLimiter(cc.batterySocLimits.Decorator()))
		implement.May(m, implement.BatteryPowerLimiter(cc.batteryPowerLimits.Decorator()))

		res, err := m.client.GetSystemStatus()
		if err != nil {
			return nil, err
		}

		implement.Has(m, implement.BatteryCapacity(func() float64 {
			return res.NominalFullPackEnergy / 1e3
		}))
	}

	if batteryControl {
		implement.May(m, implement.BatteryController(cc.batterySocLimits.LimitController(m.socG, func(limit float64) error {
			// Handle Tesla firmware 25.18.4 restrictions:
			// Values between 81-99% are not allowed, only ≤80% or exactly 100%
			limitUint := uint64(limit)
			if limitUint > 80 && limitUint < 100 {
				// Adjust to maximum allowed (80%)
				limitUint = 80
			}
			return m.energySite.SetBatteryReserve(limitUint)
		})))
	}

	return m, nil
}

// teslaEnergySite creates the fleet api energy site used for battery control
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

	// validate base url
	region, err := tc.UserRegion()
	if err != nil {
		return nil, err
	}
	tc.SetBaseUrl(region.FleetApiBaseUrl)

	if siteId == 0 {
		// auto detect energy site ID, picking first
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

// decorate soc
func (m *PowerWall) socG() (float64, error) {
	ess, err := m.energySite.EnergySiteStatus()
	if err != nil {
		return 0, err
	}
	// Fix for Tesla firmware 25.18.4: no +0.5 rounding as it interferes
	// with exact 100% reserve settings
	return math.Round(ess.PercentageCharged), nil
}
