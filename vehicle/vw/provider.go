package vw

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/thoas/go-funk"
)

// Provider implements the evcc vehicle api
type Provider struct {
	chargerG  func() (interface{}, error)
	statusG   func() (interface{}, error)
	climateG  func() (interface{}, error)
	positionG func() (interface{}, error)
	action    func(action, value string) error
	rr        func() (RolesRights, error)
}

// NewProvider provides the evcc vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		chargerG: provider.NewCached(func() (interface{}, error) {
			return api.Charger(vin)
		}, cache).InterfaceGetter(),
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
		climateG: provider.NewCached(func() (interface{}, error) {
			return api.Climater(vin)
		}, cache).InterfaceGetter(),
		positionG: provider.NewCached(func() (interface{}, error) {
			return api.Position(vin)
		}, cache).InterfaceGetter(),
		action: func(action, value string) error {
			return api.Action(vin, action, value)
		},
		rr: func() (RolesRights, error) {
			return api.RolesRights(vin)
		},
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		return float64(res.Charger.Status.BatteryStatusData.StateOfCharge.Content), nil
	}
	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		if res.Charger.Status.PlugStatusData.PlugState.Content == "connected" {
			status = api.StatusB
		}
		if res.Charger.Status.ChargingStatusData.ChargingState.Content == "charging" {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		rct := res.Charger.Status.BatteryStatusData.RemainingChargingTime

		// estimate not available
		if rct.Content == 65535 {
			return time.Time{}, api.ErrNotAvailable
		}

		timestamp, err := time.Parse(time.RFC3339, rct.Timestamp)
		return timestamp.Add(time.Duration(rct.Content) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		crsd := res.Charger.Status.CruisingRangeStatusData

		rng = int64(crsd.PrimaryEngineRange.Content)
		if crsd.EngineTypeFirstEngine.Content != "typeIsElectric" {
			rng = int64(crsd.SecondaryEngineRange.Content)
		}
	}

	return rng, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		err = api.ErrNotAvailable

		if sd := res.ServiceByID(ServiceOdometer); sd != nil {
			if fd := sd.FieldByID(ServiceOdometer); fd != nil {
				if val, err := strconv.ParseFloat(fd.Value, 64); err == nil {
					return val, nil
				}
			}
		}
	}

	return 0, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.climateG()
	if res, ok := res.(ClimaterResponse); err == nil && ok {
		state := strings.ToLower(res.Climater.Status.ClimatisationStatusData.ClimatisationState.Content)
		active := state != "off" && state != "invalid" && state != "error"

		targetTemp = res.Climater.Settings.TargetTemperature.Content
		outsideTemp = res.Climater.Status.TemperatureStatusData.OutdoorTemperature.Content
		if math.IsNaN(outsideTemp) {
			outsideTemp = targetTemp // cover "invalid"
		}

		return active, outsideTemp, targetTemp, nil
	}

	return active, outsideTemp, targetTemp, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.positionG()
	if res, ok := res.(PositionResponse); err == nil && ok {
		coord := res.FindCarResponse.Position.CarCoordinate
		return float64(coord.Latitude) / 1e6, float64(coord.Longitude) / 1e6, nil
	}

	return 0, 0, err
}

var _ api.VehicleStartCharge = (*Provider)(nil)

// StartCharge implements the api.VehicleStartCharge interface
func (v *Provider) StartCharge() error {
	return v.action(ActionCharge, ActionChargeStart)
}

var _ api.VehicleStopCharge = (*Provider)(nil)

// StopCharge implements the api.VehicleStopCharge interface
func (v *Provider) StopCharge() error {
	return v.action(ActionCharge, ActionChargeStop)
}

// var _ api.Diagnosis = (*Provider)(nil)

// Diagnose implements the api.Diagnosis interface
func (v *Provider) Diagnose2() {
	rr, err := v.rr()
	if err != nil {
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	sort.Slice(rr.OperationList.ServiceInfo, func(i, j int) bool {
		return rr.OperationList.ServiceInfo[i].ServiceId < rr.OperationList.ServiceInfo[j].ServiceId
	})

	for _, si := range rr.OperationList.ServiceInfo {
		if si.InvocationUrl.Content != "" {
			fmt.Fprintf(tw, "%s:\t%s\n", si.ServiceId, si.InvocationUrl.Content)
		}
	}

	// list remaining service
	services := funk.Map(rr.OperationList.ServiceInfo, func(si ServiceInfo) string {
		if si.InvocationUrl.Content == "" {
			return si.ServiceId
		}
		return ""
	}).([]string)

	services = funk.FilterString(services, func(s string) bool {
		return s != ""
	})

	fmt.Fprintf(tw, "without uri:\t%s\n", strings.Join(services, ","))

	tw.Flush()
}
