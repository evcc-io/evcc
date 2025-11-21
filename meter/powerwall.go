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
	"github.com/bogosj/tesla"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// PowerWall is the tesla powerwall meter
type PowerWall struct {
	usage      string
	client     *powerwall.Client
	meterG     func() (map[string]powerwall.MeterAggregatesData, error)
	energySite *tesla.EnergySite
}

func init() {
	registry.Add("tesla", NewPowerWallFromConfig)
	registry.Add("powerwall", NewPowerWallFromConfig)
}

//go:generate go tool decorate -f decoratePowerWall -b *PowerWall -r api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64" -t "api.BatterySocLimiter,GetSocLimits,func() (float64, float64)" -t "api.BatteryPowerLimiter,GetPowerLimits,func() (float64, float64)" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error"

// NewPowerWallFromConfig creates a PowerWall Powerwall Meter from generic config
func NewPowerWallFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI, Usage, User, Password string
		Cache                      time.Duration
		RefreshToken               string
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

	// support default meter names
	switch strings.ToLower(cc.Usage) {
	case "grid":
		cc.Usage = "site"
	case "pv":
		cc.Usage = "solar"
	}

	return NewPowerWall(cc.URI, cc.Usage, cc.User, cc.Password, cc.Cache, cc.RefreshToken, cc.SiteId, cc.batterySocLimits, cc.batteryPowerLimits)
}

// NewPowerWall creates a Tesla PowerWall Meter
func NewPowerWall(uri, usage, user, password string, cache time.Duration, refreshToken string, siteId int64, batterySocLimits batterySocLimits, batteryPowerLimits batteryPowerLimits) (api.Meter, error) {
	log := util.NewLogger("powerwall").Redact(user, password, refreshToken)

	httpClient := &http.Client{
		Transport: request.NewTripper(log, powerwall.DefaultTransport()),
		Timeout:   time.Second * 2, // Timeout after 2 seconds
	}

	client := powerwall.NewClient(uri, user, password, powerwall.WithHttpClient(httpClient))
	if _, err := client.GetStatus(); err != nil {
		return nil, err
	}

	m := &PowerWall{
		client: client,
		usage:  strings.ToLower(usage),
		meterG: util.Cached(client.GetMetersAggregates, cache),
	}

	var batteryControl bool
	if refreshToken != "" || siteId != 0 {
		if refreshToken == "" {
			return nil, errors.New("missing refresh token")
		}
		batteryControl = true
	}

	if batteryControl {
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))

		options := []tesla.ClientOption{tesla.WithToken(&oauth2.Token{
			RefreshToken: refreshToken,
			Expiry:       time.Now(),
		})}

		cloudClient, err := tesla.NewClient(ctx, options...)
		if err != nil {
			return nil, err
		}

		if siteId == 0 {
			// auto detect energy site ID, picking first
			products, err := cloudClient.Products()
			if err != nil {
				return nil, err
			}

			for _, p := range products {
				if p.EnergySiteId != 0 {
					siteId = p.EnergySiteId
					break
				}
			}
		}

		log.Redact(strconv.FormatInt(siteId, 10))
		energySite, err := cloudClient.EnergySite(siteId)
		if err != nil {
			return nil, err
		}
		m.energySite = energySite
	}

	// decorate api.MeterEnergy
	var totalEnergy func() (float64, error)
	if m.usage == "load" || m.usage == "solar" {
		totalEnergy = m.totalEnergy
	}

	// decorate battery
	var batteryCapacity func() float64
	var batterySoc func() (float64, error)
	var batterySocLimiter func() (float64, float64)
	var batteryPowerLimiter func() (float64, float64)

	if usage == "battery" {
		batterySoc = m.batterySoc
		batterySocLimiter = batterySocLimits.Decorator()
		batteryPowerLimiter = batteryPowerLimits.Decorator()

		res, err := m.client.GetSystemStatus()
		if err != nil {
			return nil, err
		}

		batteryCapacity = func() float64 {
			return res.NominalFullPackEnergy / 1e3
		}
	}

	// decorate api.BatteryController
	var batModeS func(api.BatteryMode) error
	if batteryControl {
		batModeS = batterySocLimits.LimitController(m.socG, func(limit float64) error {
			// Handle Tesla firmware 25.18.4 restrictions:
			// Values between 81-99% are not allowed, only ≤80% or exactly 100%
			limitUint := uint64(limit)
			if limitUint > 80 && limitUint < 100 {
				// Adjust to maximum allowed (80%)
				limitUint = 80
			}
			return m.energySite.SetBatteryReserve(limitUint)
		})
	}

	return decoratePowerWall(m, totalEnergy, batterySoc, batteryCapacity, batterySocLimiter, batteryPowerLimiter, batModeS), nil
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
	// Fix for Tesla firmware 25.18.4: Remove the problematic +0.5 rounding logic
	// that was interfering with exact 100% reserve settings. Simply return the
	// actual current SOC rounded to nearest integer.
	return math.Round(ess.PercentageCharged), nil
}
