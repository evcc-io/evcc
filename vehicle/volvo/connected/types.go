package connected

import "time"

type RechargeStatus struct {
	Status      int
	OperationID string
	Data        struct {
		BatteryChargeLevel struct {
			Value     float64 `json:",string"`
			Unit      string
			Timestamp time.Time
		}
		ElectricRange struct {
			Value     int64 `json:",string"`
			Unit      string
			Timestamp time.Time
		}
		EstimatedChargingTime struct {
			Value     int64 `json:",string"`
			Unit      string
			Timestamp time.Time
		}
		ChargingConnectionStatus struct {
			Value     string
			Timestamp time.Time
		}
		ChargingSystemStatus struct {
			Value     string
			Timestamp time.Time
		}
	}
}
