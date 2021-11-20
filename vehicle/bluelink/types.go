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
	ResMsg  StatusData
}

type StatusLatestResponse struct {
	RetCode string
	ResCode string
	ResMsg  struct {
		VehicleStatusInfo struct {
			VehicleStatus StatusData
		}
	}
}

type StatusData struct {
	Time     string
	EvStatus struct {
		BatteryStatus float64
		RemainTime2   struct {
			Atc struct {
				Value, Unit int
			}
		}
		DrvDistance []DrivingDistance
	}
	Vehicles []Vehicle
}

const (
	timeFormat = "20060102150405 -0700" // Note: must add timeOffset
	timeOffset = " +0100"
)

func (d *StatusData) Updated() (time.Time, error) {
	return time.Parse(timeFormat, d.Time+timeOffset)
}

type DrivingDistance struct {
	RangeByFuel struct {
		EvModeRange struct {
			Value int
		}
	}
}
