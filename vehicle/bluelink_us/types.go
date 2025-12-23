package bluelink_us

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

// Authentication types

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Vehicle types

type Vehicle struct {
	RegID             string `json:"regid"`
	VIN               string `json:"vin"`
	NickName          string `json:"nickName"`
	ModelCode         string `json:"modelCode"`
	VehicleGeneration string `json:"vehicleGeneration"`
	EvStatus          string `json:"evStatus"` // "E" = EV, "N" = ICE
	EnrollmentStatus  string `json:"enrollmentStatus"`
	EnrollmentDate    string `json:"enrollmentDate"`
}

type EnrollmentResponse struct {
	EnrolledVehicleDetails []struct {
		VehicleDetails Vehicle `json:"vehicleDetails"`
	} `json:"enrolledVehicleDetails"`
}

// Vehicle status types

type VehicleStatusResponse struct {
	VehicleStatus VehicleStatus `json:"vehicleStatus"`
}

type VehicleStatus struct {
	DateTime  string    `json:"dateTime"`
	EvStatus  *EvStatus `json:"evStatus,omitempty"`
	DTE       *DTE      `json:"dte,omitempty"` // distance to empty
	AirCtrlOn bool      `json:"airCtrlOn"`
	Defrost   bool      `json:"defrost"`
	Engine    bool      `json:"engine"`
	DoorLock  bool      `json:"doorLock"`
}

type EvStatus struct {
	BatteryStatus            float64           `json:"batteryStatus"`
	BatteryCharge            bool              `json:"batteryCharge"`
	BatteryPlugin            int               `json:"batteryPlugin"`
	ChargePortDoorOpenStatus int               `json:"chargePortDoorOpenStatus"`
	BatteryStndChrgPower     float64           `json:"batteryStndChrgPower,omitempty"`
	DrvDistance              []DrvDistance     `json:"drvDistance,omitempty"`
	RemainTime2              *RemainTime       `json:"remainTime2,omitempty"`
	ReservChargeInfos        *ReservChargeInfo `json:"reservChargeInfos,omitempty"`
}

type DrvDistance struct {
	RangeByFuel RangeByFuel `json:"rangeByFuel"`
}

type RangeByFuel struct {
	EvModeRange         *RangeValue `json:"evModeRange,omitempty"`
	GasModeRange        *RangeValue `json:"gasModeRange,omitempty"`
	TotalAvailableRange *RangeValue `json:"totalAvailableRange,omitempty"`
}

type RangeValue struct {
	Value float64 `json:"value"`
	Unit  int     `json:"unit"` // 1 = km, 3 = miles
}

// RemainTime contains charging time estimates
type RemainTime struct {
	Atc  TimeValue `json:"atc"`  // current charging method
	Etc1 TimeValue `json:"etc1"` // fast charging
	Etc2 TimeValue `json:"etc2"` // portable charging
	Etc3 TimeValue `json:"etc3"` // station charging
}

type TimeValue struct {
	Value int `json:"value"` // minutes
	Unit  int `json:"unit,omitempty"`
}

type ReservChargeInfo struct {
	TargetSOCList []TargetSOC `json:"targetSOClist"`
}

type TargetSOC struct {
	PlugType       int `json:"plugType"`       // 0 = DC, 1 = AC
	TargetSOCLevel int `json:"targetSOClevel"` // charge limit percentage
}

// DTE is distance to empty
type DTE struct {
	Value int `json:"value"`
	Unit  int `json:"unit"` // 1 = km, 3 = miles
}

// Position types

type PositionResponse struct {
	Coord struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
		Alt float64 `json:"alt,omitempty"`
	} `json:"coord"`
	Time string `json:"time,omitempty"`
}

const (
	plugTypeAC    = 1
	unitMiles     = 3
	kmPerMile     = 1.60934
	timeFormatISO = "2006-01-02T15:04:05Z"
)

// Interface method implementations on VehicleStatus

func (v VehicleStatus) Updated() (time.Time, error) {
	if t, err := time.Parse(timeFormatISO, v.DateTime); err == nil {
		return t, nil
	}
	return time.Time{}, api.ErrNotAvailable
}

func (v VehicleStatus) SoC() (float64, error) {
	if v.EvStatus != nil {
		return v.EvStatus.BatteryStatus, nil
	}
	return 0, api.ErrNotAvailable
}

func (v VehicleStatus) Status() (api.ChargeStatus, error) {
	if v.EvStatus != nil {
		status := api.StatusA // disconnected
		if v.EvStatus.BatteryPlugin > 0 || v.EvStatus.ChargePortDoorOpenStatus == 1 {
			status = api.StatusB // connected, not charging
		}
		if v.EvStatus.BatteryCharge {
			status = api.StatusC // charging
		}
		return status, nil
	}
	return api.StatusNone, api.ErrNotAvailable
}

func (v VehicleStatus) FinishTime() (time.Time, error) {
	if v.EvStatus != nil && v.EvStatus.RemainTime2 != nil {
		remaining := v.EvStatus.RemainTime2.Atc.Value
		if remaining > 0 {
			ts, err := v.Updated()
			if err != nil {
				ts = time.Now()
			}
			return ts.Add(time.Duration(remaining) * time.Minute), nil
		}
	}
	return time.Time{}, api.ErrNotAvailable
}

func (v VehicleStatus) Range() (int64, error) {
	if v.EvStatus != nil && len(v.EvStatus.DrvDistance) > 0 {
		if evRange := v.EvStatus.DrvDistance[0].RangeByFuel.EvModeRange; evRange != nil {
			value := evRange.Value
			if evRange.Unit == unitMiles {
				value *= kmPerMile
			}
			return int64(value + 0.5), nil
		}
	}

	// Fallback to DTE for hybrids
	if v.DTE != nil && v.DTE.Value > 0 {
		value := float64(v.DTE.Value)
		if v.DTE.Unit == unitMiles {
			value *= kmPerMile
		}
		return int64(value + 0.5), nil
	}

	return 0, api.ErrNotAvailable
}

func (v VehicleStatus) Climater() (bool, error) {
	return v.AirCtrlOn || v.Defrost, nil
}

func (v VehicleStatus) GetLimitSoc() (int64, error) {
	if v.EvStatus != nil && v.EvStatus.ReservChargeInfos != nil {
		for _, target := range v.EvStatus.ReservChargeInfos.TargetSOCList {
			if target.PlugType == plugTypeAC {
				return int64(target.TargetSOCLevel), nil
			}
		}
	}
	return 0, api.ErrNotAvailable
}
