package bluelink

import "time"

type VehiclesResponse struct {
	RetCode string
	ResMsg  struct {
		Vehicles []Vehicle
	}
}

type StatusResponse struct {
	RetCode string
	ResCode string
	ResMsg  VehicleStatus
}

type StatusLatestResponse struct {
	RetCode string
	ResCode string
	ResMsg  struct {
		VehicleStatusInfo struct {
			VehicleStatus   VehicleStatus
			VehicleLocation VehicleLocation
			Odometer        Odometer
		}
	}
}

type VehicleStatus struct {
	Time     string
	EvStatus struct {
		BatteryCharge bool
		BatteryStatus float64
		BatteryPlugin int
		RemainTime2   struct {
			Atc struct {
				Value, Unit int
			}
		}
		ChargePortDoorOpenStatus int
		DrvDistance              []DrivingDistance
		ReservChargeInfos        ReservChargeInfo
	}
	Vehicles []Vehicle
}

type VehicleLocation struct {
	Coord struct {
		Lat, Lon, Alt float64
	}
	Time string // TODO convert to timestamp
}

type Odometer struct {
	Value float64
	Unit  int
}

const (
	timeFormat = "20060102150405 -0700" // Note: must add timeOffset
	timeOffset = " +0100"

	plugTypeAC = 1
)

func (d *VehicleStatus) Updated() (time.Time, error) {
	return time.Parse(timeFormat, d.Time+timeOffset)
}

type DrivingDistance struct {
	RangeByFuel struct {
		EvModeRange struct {
			Value int
		}
	}
}

type ReservChargeInfo struct {
	TargetSocList []TargetSoc
}

type TargetSoc struct {
	TargetSocLevel int
	PlugType       int
}
