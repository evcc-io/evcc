package meter

import (
	"crypto/tls"
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

type pwsFreq struct {
	VoltageL1 *float64 `json:"ISLAND_VL1N_Main"`
	VoltageL2 *float64 `json:"ISLAND_VL2N_Main"`
	VoltageL3 *float64 `json:"ISLAND_VL3N_Main"`
}

// PowerWall is the tesla powerwall meter
type PowerWall struct {
	implement.Caps
	usage         string
	client        *powerwall.Client
	meterG        func() (map[string]powerwall.MeterAggregatesData, error)
	freqG         func() (pwsFreq, error)
	controlClient *http.Client
	controlSecret string
	controlBase   string
	opReset       func()
	minSoc        float64
	chargeMaxSoc  float64
}

type powerWallConfig struct {
	URI, Usage, User, Password string
	Scheme                     string
	SkipLogin                  bool `mapstructure:"skipLogin"`
	ControlSecret              string
	ChargeMaxSoc               float64 `mapstructure:"chargemaxsoc"`
	Insecure                   bool    `mapstructure:"insecure"`
	Cache                      time.Duration
	SiteId                     int64   // Fleet API only, deprecated for the local meter
	RefreshToken_              string  `mapstructure:"refreshToken"` // TODO deprecated
	MinSoc                     float64 // backup reserve restored in normal mode
	MaxSoc_                    any     `mapstructure:"maxsoc"`            // TODO deprecated
	MaxChargePower_            any     `mapstructure:"maxchargepower"`    // local meter override, deprecated for Fleet
	MaxDischargePower_         any     `mapstructure:"maxdischargepower"` // local meter override, deprecated for Fleet
}

func defaultPowerWallConfig() powerWallConfig {
	return powerWallConfig{
		Cache:        time.Second,
		MinSoc:       20,
		Scheme:       "https",
		ChargeMaxSoc: 100,
	}
}

// validate checks required parameters and maps legacy usage names
func (cc *powerWallConfig) validate() error {
	if cc.Usage == "" {
		return errors.New("missing usage")
	}

	if cc.Password == "" && !cc.SkipLogin {
		return errors.New("missing password")
	}

	if cc.SkipLogin {
		if cc.Scheme == "" {
			cc.Scheme = "https"
		}
		switch cc.Scheme {
		case "http", "https":
		default:
			return fmt.Errorf("invalid scheme: %s", cc.Scheme)
		}

		for name, v := range map[string]float64{"minsoc": cc.MinSoc, "chargemaxsoc": cc.ChargeMaxSoc} {
			if v < 0 || v > 100 {
				return fmt.Errorf("invalid %s: %v, must be 0-100", name, v)
			}
		}
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

	log := util.NewLogger("powerwall").Redact(cc.User, cc.Password, cc.ControlSecret)

	if cc.RefreshToken_ != "" {
		log.WARN.Println("refreshToken is deprecated, use the Powerwall (Fleet API) template for battery control")
	}

	return newPowerWall(log, cc)
}

// skipLoginTransport returns a transport for local proxies. Unlike the
// gateway transport it must not force the gateway's SNI name.
func skipLoginTransport(insecure bool) http.RoundTripper {
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		transport := base.Clone()
		if insecure {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
		return transport
	}

	if insecure {
		return &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}
	return http.DefaultTransport
}

func newPowerWall(log *util.Logger, cc powerWallConfig) (*PowerWall, error) {
	readTransport := http.RoundTripper(powerwall.DefaultTransport())
	if cc.SkipLogin {
		readTransport = skipLoginTransport(cc.Insecure)
	}

	httpClient := &http.Client{
		Transport: request.NewTripper(log, readTransport),
		Timeout:   time.Second * 2, // Timeout after 2 seconds
	}

	opts := []func(*powerwall.Client){powerwall.WithHttpClient(httpClient)}
	if cc.SkipLogin {
		if cc.Scheme != "" {
			opts = append(opts, powerwall.WithScheme(cc.Scheme))
		}
		opts = append(opts, powerwall.WithSkipLogin())
	}

	client := powerwall.NewClient(cc.URI, cc.User, cc.Password, opts...)
	if _, err := client.GetStatus(); err != nil {
		return nil, err
	}

	m := &PowerWall{
		Caps:          implement.New(),
		client:        client,
		usage:         strings.ToLower(cc.Usage),
		meterG:        util.Cached(client.GetMetersAggregates, cc.Cache),
		controlSecret: cc.ControlSecret,
		controlBase:   strings.TrimRight(cc.Scheme+"://"+cc.URI, "/"),
		minSoc:        cc.MinSoc,
		chargeMaxSoc:  cc.ChargeMaxSoc,
	}
	if cc.SkipLogin {
		m.controlClient = &http.Client{
			Transport: request.NewTripper(log, skipLoginTransport(cc.Insecure)),
			Timeout:   15 * time.Second,
		}
		m.freqG = util.Cached(m.getFreq, cc.Cache)

		if cc.ControlSecret != "" && cc.Scheme == "http" {
			log.INFO.Println("controlSecret is sent unencrypted over http; use https or a trusted network")
		}
	}

	if m.usage == "load" || m.usage == "solar" {
		implement.Has(m, implement.MeterEnergy(m.totalEnergy))
	}
	if cc.SkipLogin {
		if m.usage == "site" || m.usage == "battery" {
			implement.Has(m, implement.MeterEnergy(m.totalEnergy))
			implement.Has(m, implement.MeterReturnEnergy(m.returnEnergy))
		}
	}

	var freq pwsFreq
	freqErr := error(nil)
	if cc.SkipLogin && m.usage == "site" {
		freq, freqErr = m.freqG()
		if freqErr != nil {
			log.WARN.Println("phase readings:", freqErr)
		}
	}
	if cc.SkipLogin && freqErr == nil && m.usage == "site" &&
		freq.VoltageL1 != nil && freq.VoltageL2 != nil && freq.VoltageL3 != nil {
		implement.Has(m, implement.PhaseCurrents(m.currents))
		implement.Has(m, implement.PhaseVoltages(func() (float64, float64, float64, error) {
			res, err := m.freqG()
			if err != nil {
				return 0, 0, 0, err
			}
			if res.VoltageL1 == nil || res.VoltageL2 == nil || res.VoltageL3 == nil {
				return 0, 0, 0, errors.New("missing phase voltage")
			}
			return *res.VoltageL1, *res.VoltageL2, *res.VoltageL3, nil
		}))
	}

	if m.usage == "battery" {
		var opGet func() (*powerwall.OperationData, error)
		if cc.SkipLogin {
			opGet = m.getOperation
		} else {
			opGet = client.GetOperation
		}
		opCache := util.ResettableCached(opGet, cc.Cache)
		m.opReset = opCache.Reset

		// capacity and power limits are static, reading them validates connectivity
		status, err := client.GetSystemStatus()
		if err != nil {
			return nil, err
		}

		implement.Has(m, implement.Battery(m.batterySoc))

		implement.Has(m, implement.BatteryCapacity(func() (float64, error) {
			return status.NominalFullPackEnergy / 1e3, nil
		}))

		// backup reserve is the lower discharge limit, the powerwall has no upper soc limit
		implement.Has(m, implement.BatterySocLimiter(func() (float64, float64, error) {
			op, err := opCache.Get()
			if err != nil {
				return 0, 0, err
			}
			return op.BackupReservePercent, 100, nil
		}))

		if cc.SkipLogin {
			// The local/TEDAPI path caps max_* at the inverter usable power.
			// Take the minimum explicitly so a backend that reports the
			// nameplate in max_* cannot overstate a derated site.
			usable := minPositive(status.MaxApparentPower, status.InverterNominalUsablePower)
			charge := minPositive(status.MaxChargePower, usable)
			discharge := minPositive(status.MaxDischargePower, usable)

			if v := anyFloat(cc.MaxChargePower_); v > 0 {
				charge = v
			}
			if v := anyFloat(cc.MaxDischargePower_); v > 0 {
				discharge = v
			}

			if charge > 0 && discharge > 0 {
				// inverter apparent power applies to charging and discharging alike
				implement.Has(m, implement.BatteryPowerLimiter(func() (float64, float64, error) {
					return charge, discharge, nil
				}))
			}
		} else if status.MaxApparentPower > 0 {
			// inverter apparent power applies to charging and discharging alike
			implement.Has(m, implement.BatteryPowerLimiter(func() (float64, float64, error) {
				return status.MaxApparentPower, status.MaxApparentPower, nil
			}))
		}

		if cc.SkipLogin && cc.ControlSecret == "" {
			log.WARN.Println("no controlSecret configured, battery control disabled (read-only)")
		} else if cc.SkipLogin {
			implement.Has(m, implement.BatteryController(batteryModesSocLimit, m.setBatteryMode))
		} else if cc.ControlSecret != "" {
			log.WARN.Println("controlSecret is only supported with skipLogin, ignoring")
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
		case "load", "site", "battery":
			return float64(o.EnergyImported) / 1e3, nil
		case "solar":
			return float64(o.EnergyExported) / 1e3, nil
		}
	}

	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// returnEnergy implements the api.MeterReturnEnergy interface
func (m *PowerWall) returnEnergy() (float64, error) {
	res, err := m.meterG()
	if err != nil {
		return 0, err
	}

	if o, ok := res[m.usage]; ok {
		switch m.usage {
		case "site", "battery":
			return float64(o.EnergyExported) / 1e3, nil
		}
	}

	return 0, fmt.Errorf("invalid usage for return energy: %s", m.usage)
}

// batterySoc implements the api.Battery interface
func (m *PowerWall) batterySoc() (float64, error) {
	res, err := m.client.GetSOE()
	if err != nil {
		return 0, err
	}

	return res.Percentage, err
}
