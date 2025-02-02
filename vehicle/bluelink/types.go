package bluelink

import (
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
)

type BluelinkVehicleStatus interface {
	Updated() (time.Time, error)
	SoC() (float64, error)
	Status() (api.ChargeStatus, error)
	FinishTime() (time.Time, error)
	Range() (int64, error)
	GetLimitSoc() (int64, error)
}

type BluelinkVehicleStatusLatest interface {
	BluelinkVehicleStatus() BluelinkVehicleStatus
	Odometer() (float64, error)
	Position() (float64, float64, error)
}

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
			VehicleLocation *VehicleLocation
			Odometer        *Odometer
		}
	}
}

type VehicleStatus struct {
	Time     string
	EvStatus *struct {
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

func (d VehicleStatus) Updated() (time.Time, error) {
	return time.Parse(timeFormat, d.Time+timeOffset)
}

func (d VehicleStatus) SoC() (float64, error) {
	if d.EvStatus != nil {
		return d.EvStatus.BatteryStatus, nil
	}

	return 0, api.ErrNotAvailable
}

func (d VehicleStatus) Status() (api.ChargeStatus, error) {
	if d.EvStatus != nil {
		status := api.StatusA
		if d.EvStatus.BatteryPlugin > 0 || d.EvStatus.ChargePortDoorOpenStatus == 1 {
			status = api.StatusB
		}
		if d.EvStatus.BatteryCharge {
			status = api.StatusC
		}
		return status, nil
	}

	return api.StatusNone, api.ErrNotAvailable
}

func (d VehicleStatus) FinishTime() (time.Time, error) {
	if d.EvStatus != nil {
		remaining := d.EvStatus.RemainTime2.Atc.Value

		if remaining != 0 {
			ts, err := d.Updated()
			return ts.Add(time.Duration(remaining) * time.Minute), err
		}
	}

	return time.Time{}, api.ErrNotAvailable
}

func (d VehicleStatus) Range() (int64, error) {
	if d.EvStatus != nil {
		if dist := d.EvStatus.DrvDistance; len(dist) == 1 {
			return int64(dist[0].RangeByFuel.EvModeRange.Value), nil
		}
	}
	return 0, api.ErrNotAvailable
}

func (d VehicleStatus) GetLimitSoc() (int64, error) {
	if d.EvStatus != nil {
		for _, targetSOC := range d.EvStatus.ReservChargeInfos.TargetSocList {
			if targetSOC.PlugType == plugTypeAC {
				return int64(targetSOC.TargetSocLevel), nil
			}
		}
	}
	return 0, api.ErrNotAvailable
}

func (d StatusLatestResponse) BluelinkVehicleStatus() BluelinkVehicleStatus {
	return d.ResMsg.VehicleStatusInfo.VehicleStatus
}

func (d StatusLatestResponse) Odometer() (float64, error) {
	if d.ResMsg.VehicleStatusInfo.Odometer != nil {
		return d.ResMsg.VehicleStatusInfo.Odometer.Value, nil
	}
	return 0, api.ErrNotAvailable
}

func (d StatusLatestResponse) Position() (float64, float64, error) {
	if d.ResMsg.VehicleStatusInfo.VehicleLocation != nil {
		pos := d.ResMsg.VehicleStatusInfo.VehicleLocation.Coord
		return pos.Lat, pos.Lon, nil
	}
	return 0, 0, api.ErrNotAvailable
}

type StatusLatestResponseCCS struct {
	RetCode string
	ResCode string
	ResMsg  struct {
		State struct {
			Vehicle VehicleStatusCCS
		}
		LastUpdateTime string
	}
}

type VehicleStatusCCS struct {
	Location *struct {
		GeoCoord struct {
			Latitude, Longitude, Altitude float64
			Type                          int
			Date                          string
		}
	}
	Green *struct {
		BatteryManagement struct {
			BatteryRemain struct {
				Ratio float64
				Value float64
			}
			BatteryCapacity struct {
				Value float64
			}
			SoH struct {
				Ratio float64
			}
		}
		ChargingInformation struct {
			ConnectorFastening struct {
				// 1 connected
				State int
			}
			Charging struct {
				RemainTime     float64
				RemainTimeUnit int
			}
			EstimatedTime struct {
				Standard float64
				ICCB     float64
				Quick    float64
				Unit     int
			}
			ExpectedTime struct {
				StartDay  int
				StartHour int
				StartMin  int
				EndDay    int
				EndHour   int
				EndMin    int
			}
			TargetSoC struct {
				Standard int64
				Quick    int64
			}
			DTE struct {
				TargetSoC struct {
					// in Drivetrain.FuelSystem.DTE.Unit
					Standard float64
					Quick    float64
				}
			}
		}
		ChargingDoor struct {
			// 0, 2 closed, 1 open
			State int
		}
		Electric struct {
			SmartGrid struct {
				VehicleToLoad struct {
					DischargeLimitation struct {
						SoC        float64
						RemainTime float64
					}
				}
			}
		}
	}
	Drivetrain struct {
		Odometer   float64
		FuelSystem struct {
			DTE struct {
				Total int64
			}
		}
	}
}

func (d StatusLatestResponseCCS) BluelinkVehicleStatus() BluelinkVehicleStatus {
	return d
}

func (d StatusLatestResponseCCS) Updated() (time.Time, error) {
	epoch, err := strconv.ParseInt(d.ResMsg.LastUpdateTime, 10, 64)
	if err != nil {
		return time.Now(), err
	}
	return time.UnixMilli(epoch), nil
}

func (d StatusLatestResponseCCS) SoC() (float64, error) {
	if d.ResMsg.State.Vehicle.Green != nil {
		return d.ResMsg.State.Vehicle.Green.BatteryManagement.BatteryRemain.Ratio, nil
	}
	return 0, api.ErrNotAvailable
}

func (d StatusLatestResponseCCS) Status() (api.ChargeStatus, error) {
	if d.ResMsg.State.Vehicle.Green != nil {
		if d.ResMsg.State.Vehicle.Green.ChargingInformation.ConnectorFastening.State == 1 {
			return api.StatusB, nil
		}
		if d.ResMsg.State.Vehicle.Green.ChargingInformation.Charging.RemainTime > 0 {
			return api.StatusC, nil
		}
		return api.StatusA, nil
	}
	return api.StatusNone, api.ErrNotAvailable
}

func (d StatusLatestResponseCCS) FinishTime() (time.Time, error) {
	if d.ResMsg.State.Vehicle.Green != nil {
		remaining := d.ResMsg.State.Vehicle.Green.ChargingInformation.Charging.RemainTime

		if remaining == 0 {
			return time.Time{}, api.ErrNotAvailable
		}

		ts, err := d.Updated()
		return ts.Add(time.Duration(remaining) * time.Minute), err
	}

	return time.Now(), api.ErrNotAvailable
}

func (d StatusLatestResponseCCS) Range() (int64, error) {
	return d.ResMsg.State.Vehicle.Drivetrain.FuelSystem.DTE.Total, nil
}

func (d StatusLatestResponseCCS) GetLimitSoc() (int64, error) {
	if d.ResMsg.State.Vehicle.Green != nil {
		return d.ResMsg.State.Vehicle.Green.ChargingInformation.TargetSoC.Standard, nil
	}
	return 0, api.ErrNotAvailable
}

func (d StatusLatestResponseCCS) Odometer() (float64, error) {
	return d.ResMsg.State.Vehicle.Drivetrain.Odometer, nil
}

func (d StatusLatestResponseCCS) Position() (float64, float64, error) {
	if d.ResMsg.State.Vehicle.Location != nil {
		return d.ResMsg.State.Vehicle.Location.GeoCoord.Latitude, d.ResMsg.State.Vehicle.Location.GeoCoord.Longitude, nil
	}
	return 0, 0, api.ErrNotAvailable
}
