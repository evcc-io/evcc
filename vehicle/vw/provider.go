package vw

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/samber/lo"
)

// Provider implements the vehicle api
type Provider struct {
	chargerG  func() (ChargerResponse, error)
	statusG   func() (StatusResponse, error)
	climateG  func() (ClimaterResponse, error)
	positionG func() (PositionResponse, error)
	action    func(action, value string) error
	rr        func() (RolesRights, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		chargerG: provider.Cached(func() (ChargerResponse, error) {
			return api.Charger(vin)
		}, cache),
		statusG: provider.Cached(func() (StatusResponse, error) {
			return api.Status(vin)
		}, cache),
		climateG: provider.Cached(func() (ClimaterResponse, error) {
			return api.Climater(vin)
		}, cache),
		positionG: provider.Cached(func() (PositionResponse, error) {
			return api.Position(vin)
		}, cache),
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

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.chargerG()
	if err == nil {
		return float64(res.Charger.Status.BatteryStatusData.StateOfCharge.Content), nil
	}
	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.chargerG()
	if err == nil {
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
	if err == nil {
		rct := res.Charger.Status.BatteryStatusData.RemainingChargingTime

		// estimate not available
		if rct.Content == 65535 {
			return time.Time{}, api.ErrNotAvailable
		}

		return time.Now().Add(time.Duration(rct.Content) * time.Minute), nil
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
	res, err := v.chargerG()
	if err == nil {
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
	if err == nil {
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
func (v *Provider) Climater() (bool, error) {
	res, err := v.climateG()
	if err == nil {
		state := strings.ToLower(res.Climater.Status.ClimatisationStatusData.ClimatisationState.Content)
		active := state != "off" && state != "invalid" && state != "error"

		return active, nil
	}

	return false, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.positionG()
	if err == nil {
		coord := res.FindCarResponse.Position.CarCoordinate
		return float64(coord.Latitude) / 1e6, float64(coord.Longitude) / 1e6, nil
	}

	return 0, 0, err
}

var _ api.VehicleChargeController = (*Provider)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Provider) StartCharge() error {
	return v.action(ActionCharge, ActionChargeStart)
}

// StopCharge implements the api.VehicleChargeController interface
func (v *Provider) StopCharge() error {
	return v.action(ActionCharge, ActionChargeStop)
}

var _ api.Diagnosis = (*Provider)(nil)

// Diagnose implements the api.Diagnosis interface
func (v *Provider) Diagnose() {
	rr, err := v.rr()
	if err != nil {
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	slices.SortFunc(rr.OperationList.ServiceInfo, func(i, j ServiceInfo) int {
		return cmp.Compare(i.ServiceId, j.ServiceId)
	})

	for _, si := range rr.OperationList.ServiceInfo {
		if si.InvocationUrl.Content != "" {
			fmt.Fprintf(tw, "%s:\t%s\n", si.ServiceId, si.InvocationUrl.Content)
		}
	}

	// list remaining service
	services := lo.FilterMap(rr.OperationList.ServiceInfo, func(si ServiceInfo, _ int) (string, bool) {
		return si.ServiceId, si.InvocationUrl.Content == ""
	})

	fmt.Fprintf(tw, "without uri:\t%s\n", strings.Join(services, ","))

	tw.Flush()
}
