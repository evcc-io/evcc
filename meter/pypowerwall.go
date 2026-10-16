package meter

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// PyPowerwall is a Tesla Powerwall meter using pypowerwall-server
type PyPowerwall struct {
	implement.Caps
	usage       string
	aggregatesG func() (map[string]pypowerwallAggregate, error)
}

type pypowerwallAggregate struct {
	InstantPower   float64 `json:"instant_power"`
	EnergyImported float64 `json:"energy_imported"`
	EnergyExported float64 `json:"energy_exported"`
	CurrentA       float64 `json:"i_a_current"`
	CurrentB       float64 `json:"i_b_current"`
	CurrentC       float64 `json:"i_c_current"`
}

type pypowerwallSoe struct {
	Percentage float64 `json:"percentage"`
}

type pypowerwallSystemStatus struct {
	NominalFullPackEnergy float64 `json:"nominal_full_pack_energy"`
	MaxChargePower        float64 `json:"max_charge_power"`
	MaxDischargePower     float64 `json:"max_discharge_power"`
	MaxApparentPower      float64 `json:"max_apparent_power"`
}

// powerLimits returns the charge and discharge power limits, preferring the given
// configured values. The device reports them as max charge/discharge power or,
// failing that, as max apparent power applying to both directions. These are
// populated for Powerwall 2 but empty for Powerwall 3 (TEDAPI), which then needs
// configured limits.
func (s pypowerwallSystemStatus) powerLimits(charge, discharge float64) (float64, float64) {
	pick := func(vals ...float64) float64 {
		for _, v := range vals {
			if v > 0 {
				return v
			}
		}
		return 0
	}

	return pick(charge, s.MaxChargePower, s.MaxApparentPower),
		pick(discharge, s.MaxDischargePower, s.MaxApparentPower)
}

type pypowerwallOperation struct {
	BackupReservePercent *float64 `json:"backup_reserve_percent"`
}

type pypowerwallGrid struct {
	VoltageL1  *float64 `json:"ISLAND_VL1N_Main"`
	VoltageL2  *float64 `json:"ISLAND_VL2N_Main"`
	VoltageL3  *float64 `json:"ISLAND_VL3N_Main"`
	InverterL1 *float64 `json:"PW1_PINV_VSplit1"`
	InverterL2 *float64 `json:"PW1_PINV_VSplit2"`
}

func init() {
	registry.Add("pypowerwall", NewPyPowerwallFromConfig)
}

// NewPyPowerwallFromConfig creates a pypowerwall-server meter from generic config
func NewPyPowerwallFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI, Usage         string
		ControlSecret      string
		Insecure           bool
		MinSoc             float64 // backup reserve restored in normal mode
		ChargeMaxSoc       float64 // reserve written in charge mode
		Cache              time.Duration
		pvMaxACPower       `mapstructure:",squash"`
		batteryPowerLimits `mapstructure:",squash"`
	}{
		MinSoc:       20,
		ChargeMaxSoc: 100,
		Cache:        time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	// pypowerwall-server names
	usage, ok := map[string]string{
		"grid":    "site",
		"pv":      "solar",
		"battery": "battery",
	}[strings.ToLower(cc.Usage)]
	if !ok {
		return nil, fmt.Errorf("invalid usage: %s", cc.Usage)
	}

	log := util.NewLogger("pypowerwall").Redact(cc.ControlSecret)
	helper := request.NewHelper(log)
	if cc.Insecure {
		helper.Transport = request.NewTripper(log, transport.Insecure())
	}
	uri := strings.TrimRight(util.DefaultScheme(cc.URI, "http"), "/")

	for name, v := range map[string]float64{"minsoc": cc.MinSoc, "chargemaxsoc": cc.ChargeMaxSoc} {
		if !pypowerwallValidReserve(v) {
			return nil, fmt.Errorf("invalid %s: %v, must be 0-100", name, v)
		}
	}

	m := &PyPowerwall{
		Caps:        implement.New(),
		usage:       usage,
		aggregatesG: pypowerwallGetter[map[string]pypowerwallAggregate](helper, uri+"/api/meters/aggregates", cc.Cache),
	}

	if _, err := m.aggregate(); err != nil {
		return nil, err
	}

	// phase data requires a connection providing device vitals
	gridG := pypowerwallGetter[pypowerwallGrid](helper, uri+"/freq", cc.Cache)
	vitals, err := gridG()
	if err != nil {
		log.WARN.Println("device vitals:", err)
	}

	switch usage {
	case "site":
		implement.Has(m, implement.MeterEnergy(m.energyImported))
		implement.Has(m, implement.MeterReturnEnergy(m.energyExported))

		if vitals.VoltageL1 != nil {
			implement.Has(m, implement.PhaseCurrents(m.currents))
			implement.Has(m, implement.PhaseVoltages(func() (float64, float64, float64, error) {
				res, err := gridG()
				return floatOrZero(res.VoltageL1), floatOrZero(res.VoltageL2), floatOrZero(res.VoltageL3), err
			}))
		}

	case "solar":
		implement.Has(m, implement.MeterEnergy(m.energyExported))
		implement.May(m, implement.MaxACPowerGetter(cc.pvMaxACPower.Decorator()))

	case "battery":
		// charge and discharge energy
		implement.Has(m, implement.MeterEnergy(m.energyImported))
		implement.Has(m, implement.MeterReturnEnergy(m.energyExported))

		socG := pypowerwallGetter[pypowerwallSoe](helper, uri+"/api/system_status/soe", cc.Cache)
		soc := func() (float64, error) {
			res, err := socG()
			return res.Percentage, err
		}
		implement.Has(m, implement.Battery(soc))

		var status pypowerwallSystemStatus
		if err := helper.GetJSON(uri+"/api/system_status", &status); err != nil {
			return nil, err
		}
		if status.NominalFullPackEnergy > 0 {
			implement.Has(m, implement.BatteryCapacity(func() float64 {
				return status.NominalFullPackEnergy / 1e3
			}))
		}

		// backup reserve is the lower discharge limit, the powerwall has no upper soc limit
		operation := util.ResettableCached(func() (pypowerwallOperation, error) {
			var res pypowerwallOperation
			err := helper.GetJSON(uri+"/api/operation", &res)
			return res, err
		}, cc.Cache)

		// if unknown keep the last known reserve, never expose a lower discharge limit than the device enforces
		var mu sync.Mutex
		reserve := cc.MinSoc

		implement.Has(m, implement.BatterySocLimiter(func() (float64, float64) {
			mu.Lock()
			defer mu.Unlock()

			res, err := operation.Get()
			switch {
			case err != nil:
				log.ERROR.Println("battery soc limits:", err)
			case res.BackupReservePercent != nil:
				reserve = *res.BackupReservePercent
			}

			return reserve, 100
		}))

		if charge, discharge := status.powerLimits(cc.MaxChargePower, cc.MaxDischargePower); charge > 0 && discharge > 0 {
			implement.Has(m, implement.BatteryPowerLimiter(func() (float64, float64) {
				return charge, discharge
			}))
		}

		if vitals.InverterL1 != nil {
			implement.Has(m, implement.PhaseVoltages(func() (float64, float64, float64, error) {
				res, err := gridG()
				return floatOrZero(res.InverterL1), floatOrZero(res.InverterL2), 0, err
			}))
		}

		if cc.ControlSecret != "" {
			// charge mode raises the reserve to the charge target, normal mode restores the configured minimum
			limits := batterySocLimits{MinSoc: cc.MinSoc, MaxSoc: cc.ChargeMaxSoc}

			implement.Has(m, implement.BatteryController(batteryModesSocLimit, limits.LimitController(
				soc, pypowerwallReserveSetter(helper, uri, cc.ControlSecret, operation.Reset),
			)))
		}
	}

	return m, nil
}

// pypowerwallGetter returns a cached getter for a JSON endpoint
func pypowerwallGetter[T any](helper *request.Helper, uri string, cache time.Duration) func() (T, error) {
	return util.Cached(func() (T, error) {
		var res T
		err := helper.GetJSON(uri, &res)
		return res, err
	}, cache)
}

// pypowerwallReserveSetter returns a setter for the backup reserve using the control api
func pypowerwallReserveSetter(helper *request.Helper, uri, secret string, written func()) func(float64) error {
	auth := map[string]string{"Authorization": "Bearer " + secret}

	return func(limit float64) error {
		// the control api only accepts integer reserve levels
		data := struct {
			Value int `json:"value"`
		}{
			Value: int(math.Round(max(0, min(100, limit)))),
		}

		req, err := request.New(http.MethodPost, uri+"/control/reserve", request.MarshalJSON(data), request.JSONEncoding, auth)
		if err != nil {
			return err
		}

		if _, err := helper.DoBody(req); err != nil {
			return fmt.Errorf("set backup reserve: %w", err)
		}

		written()

		return nil
	}
}

// pypowerwallValidReserve reports whether the control api accepts the reserve.
// Local TEDAPI v1r accepts 0-100, cloud connections reject 81-99.
func pypowerwallValidReserve(v float64) bool {
	return v >= 0 && v <= 100
}

func floatOrZero(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func (m *PyPowerwall) aggregate() (pypowerwallAggregate, error) {
	res, err := m.aggregatesG()
	if err != nil {
		return pypowerwallAggregate{}, err
	}

	o, ok := res[m.usage]
	if !ok {
		return pypowerwallAggregate{}, fmt.Errorf("missing meter: %s", m.usage)
	}

	return o, nil
}

var _ api.Meter = (*PyPowerwall)(nil)

// CurrentPower implements the api.Meter interface
func (m *PyPowerwall) CurrentPower() (float64, error) {
	res, err := m.aggregate()
	return res.InstantPower, err
}

func (m *PyPowerwall) energyImported() (float64, error) {
	res, err := m.aggregate()
	return res.EnergyImported / 1e3, err
}

func (m *PyPowerwall) energyExported() (float64, error) {
	res, err := m.aggregate()
	return res.EnergyExported / 1e3, err
}

func (m *PyPowerwall) currents() (float64, float64, float64, error) {
	res, err := m.aggregate()
	return res.CurrentA, res.CurrentB, res.CurrentC, err
}
