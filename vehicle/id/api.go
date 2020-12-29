package id

import (
	"fmt"
	"net/http"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/vw"
)

// API is an api.Vehicle implementation for VW ID cars
type API struct {
	*request.Helper
	identity *vw.Identity
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity *vw.Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		identity: identity,
	}
	return v
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles() (res []string, err error) {
	uri := "https://mobileapi.apps.emea.vwapps.io/vehicles"

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + v.identity.Token(),
	})

	var vehicles struct {
		Data []struct {
			VIN string
		}
	}

	if err == nil {
		err = v.DoJSON(req, &vehicles)

		for _, v := range vehicles.Data {
			res = append(res, v.VIN)
		}
	}

	return res, err
}

// Status is the /status api
type Status struct {
	Data struct {
		BatteryStatus         BatteryStatus
		ChargingStatus        ChargingStatus
		PlugStatus            PlugStatus
		RangeStatus           RangeStatus
		ClimatisationSettings ClimatisationSettings
		// ClimatisationStatus   ClimatisationStatus // currently not available
	}
}

// BatteryStatus is the /status.batteryStatus api
type BatteryStatus struct {
	CarCapturedTimestamp    string
	CurrentSOCPercent       int `json:"currentSOC_pct"`
	CruisingRangeElectricKm int `json:"cruisingRangeElectric_km"`
}

// ChargingStatus is the /status.chargingStatus api
type ChargingStatus struct {
	CarCapturedTimestamp               string
	ChargingState                      string
	RemainingChargingTimeToCompleteMin int `json:"remainingChargingTimeToComplete_min"`
	ChargePowerKW                      int `json:"chargePower_kW"`
	ChargeRateKmph                     int `json:"chargeRate_kmph"`
}

// ChargingSettings is the /status.chargingSettings api
type ChargingSettings struct {
	CarCapturedTimestamp      string
	MaxChargeCurrentAC        string
	AutoUnlockPlugWhenCharged string
	TargetSOCPercent          int `json:"targetSOC_pct"`
}

// PlugStatus is the /status.plugStatus api
type PlugStatus struct {
	CarCapturedTimestamp string
	PlugConnectionState  string
	PlugLockState        string
}

// ClimatisationSettings is the /status.climatisationSettings api
type ClimatisationSettings struct {
	CarCapturedTimestamp              string
	TargetTemperatureK                float64 `json:"targetTemperature_K"`
	TargetTemperatureC                float64 `json:"targetTemperature_C"`
	ClimatisationWithoutExternalPower bool
	ClimatizationAtUnlock             bool
	WindowHeatingEnabled              bool
	ZoneFrontLeftEnabled              bool
	ZoneFrontRightEnabled             bool
	ZoneRearLeftEnabled               bool
	ZoneRearRightEnabled              bool
}

// RangeStatus is the /status.rangeStatus api
type RangeStatus struct {
	CarCapturedTimestamp string
	CarType              string
	PrimaryEngine        struct {
		Type              string
		CurrentSOCPercent int `json:"currentSOC_pct"`
		RemainingRangeKm  int `json:"remainingRange_km"`
	}
	TotalRangeKm int `json:"totalRange_km"`
}

// Status implements the /status response
func (v *API) Status(vin string) (res Status, err error) {
	uri := fmt.Sprintf("https://mobileapi.apps.emea.vwapps.io/vehicles/%s/status", vin)

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + v.identity.Token(),
	})

	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}
