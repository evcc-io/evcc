package meter

import (
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
	RefreshToken_              string `mapstructure:"refreshToken"` // TODO deprecated
	SiteId_                    int64  `mapstructure:"siteId"`       // TODO deprecated
	batterySocLimits           `mapstructure:",squash"`
	batteryPowerLimits         `mapstructure:",squash"`
}

func init() {
	registry.Add("tesla", NewPowerWallFromConfig)
	registry.Add("powerwall", NewPowerWallFromConfig)
}

// NewPowerWallFromConfig creates a PowerWall Powerwall Meter from generic config
func NewPowerWallFromConfig(other map[string]any) (api.Meter, error) {
	cc, err := decodePowerWallConfig(other)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("powerwall").Redact(cc.User, cc.Password)

	if cc.RefreshToken_ != "" {
		log.WARN.Println("refreshToken is deprecated, use the Powerwall (Fleet API) template for battery control")
	}

	return newPowerWall(log, cc)
}

func decodePowerWallConfig(other map[string]any) (powerWallConfig, error) {
	cc := powerWallConfig{
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
		return cc, err
	}

	if cc.Usage == "" {
		return cc, errors.New("missing usage")
	}

	if cc.Password == "" {
		return cc, errors.New("missing password")
	}

	// support default meter names
	switch strings.ToLower(cc.Usage) {
	case "grid":
		cc.Usage = "site"
	case "pv":
		cc.Usage = "solar"
	}

	return cc, nil
}

func newPowerWall(log *util.Logger, cc powerWallConfig) (*PowerWall, error) {
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
