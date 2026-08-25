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
	SiteId                     int64   // Fleet API only, deprecated for the local meter
	RefreshToken_              string  `mapstructure:"refreshToken"` // TODO deprecated
	MinSoc                     float64 // Fleet API only, backup reserve restored in normal mode
	MaxSoc_                    any     `mapstructure:"maxsoc"`            // TODO deprecated
	MaxChargePower_            any     `mapstructure:"maxchargepower"`    // TODO deprecated
	MaxDischargePower_         any     `mapstructure:"maxdischargepower"` // TODO deprecated
}

func defaultPowerWallConfig() powerWallConfig {
	return powerWallConfig{
		Cache:  time.Second,
		MinSoc: 20,
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
	registry.Add("tesla", NewPowerWallFromConfig)
	registry.Add("powerwall", NewPowerWallFromConfig)
}

// NewPowerWallFromConfig creates a PowerWall Powerwall Meter from generic config
func NewPowerWallFromConfig(other map[string]any) (api.Meter, error) {
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

	return newPowerWall(log, cc)
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
		opG := util.Cached(client.GetOperation, cc.Cache)

		// capacity and power limits are static, reading them validates connectivity
		status, err := client.GetSystemStatus()
		if err != nil {
			return nil, err
		}

		implement.Has(m, implement.Battery(m.batterySoc))

		implement.Has(m, implement.BatteryCapacity(func() float64 {
			return status.NominalFullPackEnergy / 1e3
		}))

		// backup reserve is the lower discharge limit, the powerwall has no upper soc limit
		implement.Has(m, implement.BatterySocLimiter(func() (float64, float64) {
			op, err := opG()
			if err != nil {
				log.ERROR.Println("battery soc limits:", err)
				return 0, 100
			}
			return op.BackupReservePercent, 100
		}))

		if status.MaxApparentPower > 0 {
			// inverter apparent power applies to charging and discharging alike
			implement.Has(m, implement.BatteryPowerLimiter(func() (float64, float64) {
				return status.MaxApparentPower, status.MaxApparentPower
			}))
		}
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
