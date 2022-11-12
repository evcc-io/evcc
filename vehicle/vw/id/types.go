package id

import (
	"fmt"
	"strings"
	"time"
)

// Status is the /status api
type Status struct {
	Data struct {
		BatteryStatus         `json:"batteryStatus"`
		ChargingStatus        `json:"chargingStatus"`
		ChargingSettings      `json:"chargingSettings"`
		PlugStatus            `json:"plugStatus"`
		RangeStatus           `json:"rangeStatus"`
		ClimatisationSettings `json:"climatisationSettings"`
		ClimatisationStatus   `json:"climatisationStatus"` // may be temporarily not available
		*MaintenanceStatus    `json:"maintenanceStatus"`   // optional
	}
	Error ErrorMap
}

type ErrorMap map[string]Error

func (e ErrorMap) Extract(key string) error {
	for k, v := range e {
		if k == key {
			return fmt.Errorf("%s: %s", v.Info, v.Message)
		}
	}

	return nil
}

// Error is the error status
type Error struct {
	Code          int
	Message, Info string
}

// BatteryStatus is the /status.batteryStatus api
type BatteryStatus struct {
	CarCapturedTimestamp    Timestamp
	CurrentSOCPercent       int `json:"currentSOC_pct"`
	CruisingRangeElectricKm int `json:"cruisingRangeElectric_km"`
}

// ChargingStatus is the /status.chargingStatus api
type ChargingStatus struct {
	CarCapturedTimestamp               Timestamp
	ChargingState                      string  // readyForCharging/off
	ChargeMode                         string  // invalid
	RemainingChargingTimeToCompleteMin int     `json:"remainingChargingTimeToComplete_min"`
	ChargePowerKW                      float64 `json:"chargePower_kW"`
	ChargeRateKmph                     float64 `json:"chargeRate_kmph"`
}

// ChargingSettings is the /status.chargingSettings api
type ChargingSettings struct {
	CarCapturedTimestamp      Timestamp
	MaxChargeCurrentAC        string // reduced, maximum
	AutoUnlockPlugWhenCharged string
	TargetSOCPercent          int `json:"targetSOC_pct"`
}

// PlugStatus is the /status.plugStatus api
type PlugStatus struct {
	CarCapturedTimestamp Timestamp
	PlugConnectionState  string // connected, disconnected
	PlugLockState        string // locked, unlocked
}

// ClimatisationStatus is the /status.climatisationStatus api
type ClimatisationStatus struct {
	CarCapturedTimestamp          Timestamp
	RemainingClimatisationTimeMin int    `json:"remainingClimatisationTime_min"`
	ClimatisationState            string // off
}

// ClimatisationSettings is the /status.climatisationSettings api
type ClimatisationSettings struct {
	CarCapturedTimestamp              Timestamp
	TargetTemperatureK                float64 `json:"targetTemperature_K"`
	TargetTemperatureC                float64 `json:"targetTemperature_C"`
	ClimatisationWithoutExternalPower bool
	ClimatisationAtUnlock             bool // ClimatizationAtUnlock?
	WindowHeatingEnabled              bool
	ZoneFrontLeftEnabled              bool
	ZoneFrontRightEnabled             bool
	ZoneRearLeftEnabled               bool
	ZoneRearRightEnabled              bool
}

// RangeStatus is the /status.rangeStatus api
type RangeStatus struct {
	CarCapturedTimestamp Timestamp
	CarType              string
	PrimaryEngine        struct {
		Type              string
		CurrentSOCPercent int `json:"currentSOC_pct"`
		RemainingRangeKm  int `json:"remainingRange_km"`
	}
	TotalRangeKm int `json:"totalRange_km"`
}

// MaintenanceStatus is the /status.maintenanceStatus api
type MaintenanceStatus struct {
	CarCapturedTimestamp Timestamp // "2021-09-04T14:59:10Z",
	InspectionDueDays    int       `json:"inspectionDue_days"`
	InspectionDueKm      int       `json:"inspectionDue_km"`
	MileageKm            int       `json:"mileage_km"`
	OilServiceDueDays    int       `json:"oilServiceDue_days"`
	OilServiceDueKm      int       `json:"oilServiceDue_km"`
}

// Timestamp implements JSON unmarshal
type Timestamp struct {
	time.Time
}

// UnmarshalJSON decodes string timestamp into time.Time
func (ct *Timestamp) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")

	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		(*ct).Time = t
	}

	return err
}
