package tesla

import "errors"

type Region struct {
	Region          string
	FleetApiBaseUrl string `json:"fleet_api_base_url"`
}

type Vehicle struct {
	DisplayName string `json:"display_name"`
	Vin         string `json:"vin"`
}

type VehicleData struct {
	Response struct {
		ChargeState struct {
			ChargingState      string  `json:"charging_state"`
			UsableBatteryLevel int     `json:"usable_battery_level"`
			ChargeEnergyAdded  float64 `json:"charge_energy_added"`
			BatteryRange       float64 `json:"battery_range"`
			ChargeLimitSoc     int     `json:"charge_limit_soc"`
		} `json:"charge_state"`
		ClimateState struct {
			IsPreconditioning bool `json:"is_preconditioning"`
		} `json:"climate_state"`
		DriveState struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"drive_state"`
		VehicleState struct {
			Odometer float64 `json:"odometer"`
		} `json:"vehicle_state"`
	}
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// CommandResponse covers vehicle commands (reason/result) and energy site commands (code/message)
type CommandResponse struct {
	Response struct {
		Reason  string `json:"reason"`
		Result  bool   `json:"result"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	Error string `json:"error"`
}

// Reason returns the command failure as error
func (res CommandResponse) Reason() error {
	if !res.Response.Result && res.Response.Reason != "" {
		return errors.New(res.Response.Reason)
	}
	if res.Response.Code != 0 && res.Response.Code != 200 && res.Response.Code != 201 {
		return errors.New(res.Response.Message)
	}
	if res.Error != "" {
		return errors.New(res.Error)
	}
	return nil
}

type Product struct {
	EnergySiteId int64 `json:"energy_site_id"`
}

type EnergySiteStatus struct {
	PercentageCharged float64 `json:"percentage_charged"`
}

type EnergySiteInfo struct {
	Components struct {
		DisallowChargeFromGridWithSolarInstalled bool `json:"disallow_charge_from_grid_with_solar_installed"`
	}
}
