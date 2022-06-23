package carwings

import (
	"encoding/json"
	"time"
)

const (
	carwingsStatusExpiry   = 5 * time.Minute // if returned status value is older, evcc will init refresh
	carwingsRefreshTimeout = 2 * time.Minute // timeout to get status after refresh
)

type Connect struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Baseprm string `json:"baseprm"`
}

type LoginResponse struct {
	StatusCode      int           `json:"status"`
	Message         string        `json:"message"`
	VehicleInfos    []VehicleInfo `json:"vehicleInfo"`
	VehicleInfoList struct {
		VehicleInfos []VehicleInfo `json:"vehicleInfo"`
	} `json:"vehicleInfoList"`
	VehicleInfo VehicleInfo `json:"VehicleInfo"`

	CustomerInfo struct {
		Timezone    string
		VehicleInfo VehicleInfo `json:"VehicleInfo"`
	}
}

type VehicleInfo struct {
	VIN             string `json:"vin"`
	CustomSessionID string `json:"custom_sessionid"`
}

type ChargerResponse struct {
	StatusCode          int    `json:"status"`
	Message             string `json:"message"`
	BatteryStatusRecord struct {
		BatteryStatus struct {
			BatteryChargingStatus     string
			BatteryCapacity           int `json:",string"`
			BatteryRemainingAmount    string
			BatteryRemainingAmountWH  string
			BatteryRemainingAmountKWH string
			SOC                       struct {
				Value int `json:",string"`
			}
		}
		PluginState        string
		CruisingRangeAcOn  json.Number `json:",string"`
		CruisingRangeAcOff json.Number `json:",string"`
		TimeRequiredToFull struct {
			HourRequiredToFull    int `json:",string"`
			MinutesRequiredToFull int `json:",string"`
		}
		TimeRequiredToFull200 struct {
			HourRequiredToFull    int `json:",string"`
			MinutesRequiredToFull int `json:",string"`
		}
		TimeRequiredToFull200_6kW struct {
			HourRequiredToFull    int `json:",string"`
			MinutesRequiredToFull int `json:",string"`
		}
		NotificationDateAndTime string
	}
}

type ClimaterResponse struct {
	StatusCode      int    `json:"status"`
	Message         string `json:"message"`
	RemoteACRecords struct {
		OperationResult        string
		OperationDateAndTime   time.Time
		RemoteACOperation      string
		ACStartStopDateAndTime time.Time
		ACStartStopURL         string
		CruisingRangeAcOn      json.Number `json:",string"`
		CruisingRangeAcOff     json.Number `json:",string"`
		PluginState            string
		ACDurationBatterySec   int `json:",string"`
		ACDurationPluggedSec   int `json:",string"`
		PreAC_unit             string
		PreAC_temp             int `json:",string"`
	}
}

type UpdateResponse struct {
	StatusCode int    `json:"status"`
	Message    string `json:"message"`
	ResultKey  string `json:"resultKey"`
}

type CheckUpdateResponse struct {
	StatusCode      int    `json:"status"`
	Message         string `json:"message"`
	ResponseFlag    int    `json:"responseFlag,string"`
	OperationResult string `json:"operationResult"`
}
