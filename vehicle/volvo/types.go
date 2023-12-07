package volvo

import (
	"strings"
	"time"
)

const ApiURI = "https://vocapi.wirelesscar.net/customerapi/rest/v3.0"

type AccountResponse struct {
	ErrorLabel       string   `json:"errorLabel"`
	ErrorDescription string   `json:"errorDescription"`
	FirstName        string   `json:"firstName"`
	LastName         string   `json:"lastName"`
	VehicleRelations []string `json:"accountVehicleRelations"`
}

type VehicleRelation struct {
	Account                   string `json:"account"`
	AccountID                 string `json:"accountId"`
	Vehicle                   string `json:"vehicle"`
	AccountVehicleRelation    string `json:"accountVehicleRelation"`
	VehicleID                 string `json:"vehicleId"`
	Username                  string `json:"username"`
	Status                    string `json:"status"`
	CustomerVehicleRelationID int    `json:"customerVehicleRelationId"`
}

type Status struct {
	ErrorLabel                      string    `json:"errorLabel"`
	ErrorDescription                string    `json:"errorDescription"`
	AverageFuelConsumption          float32   `json:"averageFuelConsumption"`
	AverageFuelConsumptionTimestamp Timestamp `json:"averageFuelConsumptionTimestamp"`
	AverageSpeed                    int       `json:"averageSpeed"`
	AverageSpeedTimestamp           Timestamp `json:"averageSpeedTimestamp"`
	BrakeFluid                      string    `json:"brakeFluid"`
	BrakeFluidTimestamp             Timestamp `json:"brakeFluidTimestamp"`
	CarLocked                       bool      `json:"carLocked"`
	CarLockedTimestamp              Timestamp `json:"carLockedTimestamp"`
	ConnectionStatus                string    `json:"connectionStatus"` // Disconnected
	ConnectionStatusTimestamp       Timestamp `json:"connectionStatusTimestamp"`
	DistanceToEmpty                 int       `json:"distanceToEmpty"`
	DistanceToEmptyTimestamp        Timestamp `json:"distanceToEmptyTimestamp"`
	EngineRunning                   bool      `json:"engineRunning"`
	EngineRunningTimestamp          Timestamp `json:"engineRunningTimestamp"`
	FuelAmount                      int       `json:"fuelAmount"`
	FuelAmountLevel                 int       `json:"fuelAmountLevel"`
	FuelAmountLevelTimestamp        Timestamp `json:"fuelAmountLevelTimestamp"`
	FuelAmountTimestamp             Timestamp `json:"fuelAmountTimestamp"`
	HvBattery                       struct {
		HvBatteryChargeStatusDerived          string    `json:"hvBatteryChargeStatusDerived"` // CableNotPluggedInCar, CablePluggedInCar, Charging
		HvBatteryChargeStatusDerivedTimestamp Timestamp `json:"hvBatteryChargeStatusDerivedTimestamp"`
		HvBatteryChargeModeStatus             string    `json:"hvBatteryChargeModeStatus"`
		HvBatteryChargeModeStatusTimestamp    Timestamp `json:"hvBatteryChargeModeStatusTimestamp"`
		HvBatteryChargeStatus                 string    `json:"hvBatteryChargeStatus"` // Started, ChargeProgress, ChargeEnd, Interrupted
		HvBatteryChargeStatusTimestamp        Timestamp `json:"hvBatteryChargeStatusTimestamp"`
		HvBatteryLevel                        int       `json:"hvBatteryLevel"`
		HvBatteryLevelTimestamp               Timestamp `json:"hvBatteryLevelTimestamp"`
		DistanceToHVBatteryEmpty              int       `json:"distanceToHVBatteryEmpty"`
		DistanceToHVBatteryEmptyTimestamp     Timestamp `json:"distanceToHVBatteryEmptyTimestamp"`
		TimeToHVBatteryFullyCharged           int       `json:"timeToHVBatteryFullyCharged"`
		TimeToHVBatteryFullyChargedTimestamp  Timestamp `json:"timeToHVBatteryFullyChargedTimestamp"`
	} `json:"hvBattery"`
	Odometer                           float64   `json:"odometer"`
	OdometerTimestamp                  Timestamp `json:"odometerTimestamp"`
	PrivacyPolicyEnabled               bool      `json:"privacyPolicyEnabled"`
	PrivacyPolicyEnabledTimestamp      Timestamp `json:"privacyPolicyEnabledTimestamp"`
	RemoteClimatizationStatus          string    `json:"remoteClimatizationStatus"` // CableConnectedWithoutPower
	RemoteClimatizationStatusTimestamp Timestamp `json:"remoteClimatizationStatusTimestamp"`
	ServiceWarningStatus               string    `json:"serviceWarningStatus"`
	ServiceWarningStatusTimestamp      Timestamp `json:"serviceWarningStatusTimestamp"`
	TimeFullyAccessibleUntil           string    `json:"timeFullyAccessibleUntil"`
	TimePartiallyAccessibleUntil       string    `json:"timePartiallyAccessibleUntil"`
	TripMeter1                         int       `json:"tripMeter1"`
	TripMeter1Timestamp                Timestamp `json:"tripMeter1Timestamp"`
	TripMeter2                         int       `json:"tripMeter2"`
	TripMeter2Timestamp                Timestamp `json:"tripMeter2Timestamp"`
}

const timeFormat = "2006-01-02T15:04:05-0700"

// Timestamp implements JSON unmarshal
type Timestamp struct {
	time.Time
}

// UnmarshalJSON decodes string timestamp into time.Time
func (ct *Timestamp) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")

	if s == "null" {
		return nil
	}

	t, err := time.Parse(timeFormat, s)
	if err == nil {
		ct.Time = t
	}

	return err
}
