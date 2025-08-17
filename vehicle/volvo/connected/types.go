package connected

import "time"

type EnergyState struct {
	BatteryChargeLevel struct {
		Status    string
		Value     float64
		Unit      string
		Timestamp time.Time
	}
	ElectricRange struct {
		Status    string
		Value     int64
		Unit      string
		Timestamp time.Time
	}
	ChargerConnectionStatus struct {
		Status    string
		Value     string
		Timestamp time.Time
	}
	ChargingStatus struct {
		Status    string
		Value     string
		Timestamp time.Time
	}
	ChargingType struct {
		Status    string
		Value     string
		Timestamp time.Time
	}
	ChargerPowerStatus struct {
		Status    string
		Value     string
		Timestamp time.Time
	}
	EstimatedChargingTimeTimeToTargetBatteryChargeLevel struct {
		Status    string
		Value     int64
		Unit      string
		Timestamp time.Time
	}
	ChargingCurrentLimit struct {
		Status    string
		Value     int64
		Unit      string
		Timestamp time.Time
	}
	TargetBatteryChargeLevel struct {
		Status    string
		Value     float64
		Unit      string
		Timestamp time.Time
	}
	ChargingPower struct {
		Status    string
		Value     int64
		Unit      string
		Timestamp time.Time
	}
}

type Vehicle struct {
	VIN string
}

type OdometerState struct {
	Data struct {
		Odometer struct {
			Status    string
			Value     int64
			Unit      string
			Timestamp time.Time
		}
	}
}
