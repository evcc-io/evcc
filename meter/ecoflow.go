package meter

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/spf13/cast"
	"github.com/tess1o/go-ecoflow"
)

// EcoFlow represents the EcoFlow  meter
type EcoFlow struct {
	implement.Caps
	log    *util.Logger
	usage  string
	serial string
	cache  time.Duration
	client *ecoflow.Client
	dataG  func() (*ecoflow.GetCmdResponse, error)
	limitG func() (float64, error)

	power, batterySoc string
}

func init() {
	registry.Add("ecoflow", NewEcoFlowFromConfig)
}

// NewEcoFlowFromConfig creates an EcoFlow  meter from generic config
func NewEcoFlowFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		batteryCapacity                      `mapstructure:",squash"`
		batteryPowerLimits                   `mapstructure:",squash"`
		batterySocLimits                     `mapstructure:",squash"`
		Usage                                string
		AccessKey, SecretKey, Serial, Region string
		Power, Soc                           string
		Cache                                time.Duration
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.AccessKey == "" {
		return nil, errors.New("missing access key")
	}
	if cc.SecretKey == "" {
		return nil, errors.New("missing secret key")
	}
	if cc.Serial == "" {
		return nil, errors.New("missing serial")
	}
	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	var uri string
	switch cc.Region {
	case "auto":
		uri = "https://api.ecoflow.com"
	case "europe":
		uri = "https://api-e.ecoflow.com"
	case "america":
		uri = "https://api-a.ecoflow.com"
	default:
		return nil, fmt.Errorf("invalid region: %s", cc.Region)
	}

	return NewEcoFlow(cc.AccessKey, cc.SecretKey, cc.Serial, cc.Usage, uri, cc.Power, cc.Soc, cc.Cache, cc.batteryCapacity.Decorator(), cc.batterySocLimits, cc.batteryPowerLimits.Decorator())
}

// NewEcoFlow constructs the EcoFlow struct
func NewEcoFlow(accessKey, secretKey, serial, usage, uri string,
	power, soc string, cache time.Duration, capacity func() float64, batterySocLimits batterySocLimits, batteryPowerLimits func() (float64, float64)) (*EcoFlow, error) {
	log := util.NewLogger("ecoflow").Redact(accessKey, secretKey, serial)

	m := &EcoFlow{
		Caps:   implement.New(),
		log:    log,
		serial: serial,
		usage:  usage,
		cache:  cache,
		client: ecoflow.NewEcoflowClient(accessKey, secretKey,
			ecoflow.WithBaseUrl(uri),
			ecoflow.WithHttpClient(request.NewClient(log)),
		),
		power:      power,
		batterySoc: soc,
	}

	m.dataG = util.Cached(m.getData, cache)

	if usage == "battery" {
		implement.Has(m, implement.Battery(m.soc))
		implement.May(m, implement.BatteryCapacity(capacity))
		implement.May(m, implement.BatterySocLimiter(batterySocLimits.Decorator()))
		implement.May(m, implement.BatteryPowerLimiter(batteryPowerLimits))

		// the backup reserve command is Stream-specific, the PowerOcean template shares this meter type
		if soc == "cmsBattSoc" && batterySocLimits.MaxSoc > 0 {
			m.limitG = util.Cached(m.dischargeLimit, cache)
			implement.Has(m, implement.BatteryController(batteryModesSocLimit, batterySocLimits.LimitController(m.soc, m.setBackupReserve)))
		}
	}

	return m, nil
}

// getData retrieves device parameters from EcoFlow API
func (m *EcoFlow) getData() (*ecoflow.GetCmdResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := []string{m.power}

	if m.usage == "battery" {
		params = append(params, m.batterySoc)
	}

	return m.client.GetDeviceParameters(ctx, m.serial, params)
}

var _ api.Meter = (*EcoFlow)(nil)

// CurrentPower implements the api.Meter interface
func (m *EcoFlow) CurrentPower() (float64, error) {
	response, err := m.dataG()
	if err != nil {
		return 0, err
	}

	pwr, err := ecoflowValue(response.Data, m.power)
	if err != nil {
		return 0, err
	}

	if m.usage == "battery" {
		pwr = -pwr // invert battery power: ecoflow returns negative when discharging and positive when charging.
	}

	return pwr, nil
}

// extractFloat extracts a float64 or int value from a map by key.
func ecoflowValue(data map[string]any, key string) (float64, error) {
	if data != nil {
		if v, ok := data[key]; ok {
			return cast.ToFloat64E(v)
		}
	}
	return 0, api.ErrNotAvailable
}

// soc returns the battery state of charge
func (m *EcoFlow) soc() (float64, error) {
	response, err := m.dataG()
	if err != nil {
		return 0, err
	}

	return ecoflowValue(response.Data, m.batterySoc)
}

// ecoflowReserveLimit clamps the reserve above the discharge limit, which the device requires it
// to exceed by 3
func ecoflowReserveLimit(limit, lo float64) float64 {
	return min(100, max(limit, lo+3))
}

// dischargeLimit reads the device's discharge limit (cmsMinDsgSoc), set in the EcoFlow app and
// polled separately from getData so a rejected quota can't degrade the power and soc readings
func (m *EcoFlow) dischargeLimit() (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := m.client.GetDeviceParameters(ctx, m.serial, []string{"cmsMinDsgSoc"})
	if err != nil {
		return 0, err
	}

	return ecoflowValue(res.Data, "cmsMinDsgSoc")
}

func (m *EcoFlow) setBackupReserve(limit float64) error {
	lo, err := m.limitG()
	if err != nil {
		return fmt.Errorf("discharge limit: %w", err)
	}

	// device requires the reserve to exceed the discharge limit by 3
	if clamped := ecoflowReserveLimit(limit, lo); clamped != limit {
		m.log.DEBUG.Printf("backup reserve: raised %.0f to %.0f, below discharge limit %.0f", limit, clamped, lo)
		limit = clamped
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// cfgBackupReverseSoc is the vendor's spelling of the backup reserve
	res, err := m.client.SetDeviceParameter(ctx, map[string]any{
		"sn": m.serial, "cmdId": 17, "cmdFunc": 254,
		"dirDest": 1, "dirSrc": 1, "dest": 2, "needAck": true,
		"params": map[string]any{"cfgBackupReverseSoc": int(math.Round(limit))},
	})
	if err != nil {
		return err
	}
	if res == nil {
		return errors.New("invalid response")
	}
	if res.Code != "0" {
		return fmt.Errorf("%s (%s)", res.Message, res.Code)
	}
	return nil
}
