package volvo

const ApiURI = "https://vocapi.wirelesscar.net/customerapi/rest/v3.0"

type AccountResponse struct {
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
	AverageFuelConsumption          float32 `json:"averageFuelConsumption"`
	AverageFuelConsumptionTimestamp string  `json:"averageFuelConsumptionTimestamp"`
	AverageSpeed                    int     `json:"averageSpeed"`
	AverageSpeedTimestamp           string  `json:"averageSpeedTimestamp"`
	BrakeFluid                      string  `json:"brakeFluid"`
	BrakeFluidTimestamp             string  `json:"brakeFluidTimestamp"`
	CarLocked                       bool    `json:"carLocked"`
	CarLockedTimestamp              string  `json:"carLockedTimestamp"`
	ConnectionStatus                string  `json:"connectionStatus"` // Disconnected
	ConnectionStatusTimestamp       string  `json:"connectionStatusTimestamp"`
	DistanceToEmpty                 int     `json:"distanceToEmpty"`
	DistanceToEmptyTimestamp        string  `json:"distanceToEmptyTimestamp"`
	EngineRunning                   bool    `json:"engineRunning"`
	EngineRunningTimestamp          string  `json:"engineRunningTimestamp"`
	FuelAmount                      int     `json:"fuelAmount"`
	FuelAmountLevel                 int     `json:"fuelAmountLevel"`
	FuelAmountLevelTimestamp        string  `json:"fuelAmountLevelTimestamp"`
	FuelAmountTimestamp             string  `json:"fuelAmountTimestamp"`
	HvBattery                       struct {
		HvBatteryChargeStatusDerived          string `json:"hvBatteryChargeStatusDerived"` // CableNotPluggedInCar, CablePluggedInCar, Charging
		HvBatteryChargeStatusDerivedTimestamp string `json:"hvBatteryChargeStatusDerivedTimestamp"`
		HvBatteryChargeModeStatus             string `json:"hvBatteryChargeModeStatus"`
		HvBatteryChargeModeStatusTimestamp    string `json:"hvBatteryChargeModeStatusTimestamp"`
		HvBatteryChargeStatus                 string `json:"hvBatteryChargeStatus"` // Started, ChargeProgress, ChargeEnd, Interrupted
		HvBatteryChargeStatusTimestamp        string `json:"hvBatteryChargeStatusTimestamp"`
		HvBatteryLevel                        int    `json:"hvBatteryLevel"`
		HvBatteryLevelTimestamp               string `json:"hvBatteryLevelTimestamp"`
		DistanceToHVBatteryEmpty              int    `json:"distanceToHVBatteryEmpty"`
		DistanceToHVBatteryEmptyTimestamp     string `json:"distanceToHVBatteryEmptyTimestamp"`
		TimeToHVBatteryFullyCharged           int    `json:"timeToHVBatteryFullyCharged"`
		TimeToHVBatteryFullyChargedTimestamp  string `json:"timeToHVBatteryFullyChargedTimestamp"`
	} `json:"hvBattery"`
	Odometer                           float64 `json:"odometer"`
	OdometerTimestamp                  string  `json:"odometerTimestamp"`
	PrivacyPolicyEnabled               bool    `json:"privacyPolicyEnabled"`
	PrivacyPolicyEnabledTimestamp      string  `json:"privacyPolicyEnabledTimestamp"`
	RemoteClimatizationStatus          string  `json:"remoteClimatizationStatus"` // CableConnectedWithoutPower
	RemoteClimatizationStatusTimestamp string  `json:"remoteClimatizationStatusTimestamp"`
	ServiceWarningStatus               string  `json:"serviceWarningStatus"`
	ServiceWarningStatusTimestamp      string  `json:"serviceWarningStatusTimestamp"`
	TimeFullyAccessibleUntil           string  `json:"timeFullyAccessibleUntil"`
	TimePartiallyAccessibleUntil       string  `json:"timePartiallyAccessibleUntil"`
	TripMeter1                         int     `json:"tripMeter1"`
	TripMeter1Timestamp                string  `json:"tripMeter1Timestamp"`
	TripMeter2                         int     `json:"tripMeter2"`
	TripMeter2Timestamp                string  `json:"tripMeter2Timestamp"`
}
