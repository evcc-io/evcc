package id

import (
	"strings"
	"time"
)

// Status is the /status api
type Status struct {
	Automation *struct {
		ClimatisationTimer struct {
			Value struct {
				Timers []struct {
					ID             int  `json:"id"`
					Enabled        bool `json:"enabled"`
					RecurringTimer struct {
						StartTime   string `json:"startTime"`
						RecurringOn struct {
							Mondays    bool `json:"mondays"`
							Tuesdays   bool `json:"tuesdays"`
							Wednesdays bool `json:"wednesdays"`
							Thursdays  bool `json:"thursdays"`
							Fridays    bool `json:"fridays"`
							Saturdays  bool `json:"saturdays"`
							Sundays    bool `json:"sundays"`
						} `json:"recurringOn"`
					} `json:"recurringTimer,omitempty"`
					SingleTimer struct {
						StartDateTime Timestamp `json:"startDateTime"`
					} `json:"singleTimer,omitempty"`
				} `json:"timers"`
				CarCapturedTimestamp Timestamp `json:"carCapturedTimestamp"`
				TimeInCar            Timestamp `json:"timeInCar"`
			} `json:"value"`
		} `json:"climatisationTimer"`
		ChargingProfiles *struct {
			Value struct {
				CarCapturedTimestamp Timestamp     `json:"carCapturedTimestamp"`
				TimeInCar            Timestamp     `json:"timeInCar"`
				Profiles             []interface{} `json:"profiles"`
			} `json:"value"`
		} `json:"chargingProfiles"`
	} `json:"automation"`
	UserCapabilities *struct {
		CapabilitiesStatus struct {
			Value []struct {
				ID                   string    `json:"id"`
				UserDisablingAllowed bool      `json:"userDisablingAllowed"`
				ExpirationDate       Timestamp `json:"expirationDate,omitempty"`
				Status               []int     `json:"status,omitempty"`
			} `json:"value"`
		} `json:"capabilitiesStatus"`
	} `json:"userCapabilities"`
	Charging *struct {
		BatteryStatus struct {
			Value struct {
				CarCapturedTimestamp    Timestamp `json:"carCapturedTimestamp"`
				CurrentSOCPct           int       `json:"currentSOC_pct"`
				CruisingRangeElectricKm int       `json:"cruisingRangeElectric_km"`
			} `json:"value"`
		} `json:"batteryStatus"`
		ChargingStatus struct {
			Value struct {
				CarCapturedTimestamp               Timestamp `json:"carCapturedTimestamp"`
				RemainingChargingTimeToCompleteMin int       `json:"remainingChargingTimeToComplete_min"`
				ChargingState                      string    `json:"chargingState"` // readyForCharging/off
				ChargeMode                         string    `json:"chargeMode"`    // invalid
				ChargePowerKW                      int       `json:"chargePower_kW"`
				ChargeRateKmph                     int       `json:"chargeRate_kmph"`
				ChargeType                         string    `json:"chargeType"`
				ChargingSettings                   string    `json:"chargingSettings"`
			} `json:"value"`
		} `json:"chargingStatus"`
		ChargingSettings struct {
			Value struct {
				CarCapturedTimestamp        Timestamp `json:"carCapturedTimestamp"`
				MaxChargeCurrentAC          string    `json:"maxChargeCurrentAC"` // reduced, maximum
				AutoUnlockPlugWhenCharged   string    `json:"autoUnlockPlugWhenCharged"`
				AutoUnlockPlugWhenChargedAC string    `json:"autoUnlockPlugWhenChargedAC"`
				TargetSOCPct                int       `json:"targetSOC_pct"`
			} `json:"value"`
		} `json:"chargingSettings"`
		PlugStatus struct {
			Value struct {
				CarCapturedTimestamp Timestamp `json:"carCapturedTimestamp"`
				PlugConnectionState  string    `json:"plugConnectionState"` // connected, disconnected
				PlugLockState        string    `json:"plugLockState"`       // locked, unlocked
				ExternalPower        string    `json:"externalPower"`
				LedColor             string    `json:"ledColor"`
			} `json:"value"`
		} `json:"plugStatus"`
		ChargeMode struct {
			Value struct {
				PreferredChargeMode  string   `json:"preferredChargeMode"`
				AvailableChargeModes []string `json:"availableChargeModes"`
			} `json:"value"`
		} `json:"chargeMode"`
	} `json:"charging"`
	Climatisation *struct {
		ClimatisationStatus struct {
			Value struct {
				CarCapturedTimestamp          Timestamp `json:"carCapturedTimestamp"`
				RemainingClimatisationTimeMin int       `json:"remainingClimatisationTime_min"`
				ClimatisationState            string    `json:"climatisationState"` // off
			} `json:"value"`
		} `json:"climatisationStatus"` // may be temporarily not available
		ClimatisationSettings struct {
			Value struct {
				CarCapturedTimestamp  Timestamp `json:"carCapturedTimestamp"`
				TargetTemperatureC    float64   `json:"targetTemperature_C"`
				TargetTemperatureF    float64   `json:"targetTemperature_F"`
				UnitInCar             string    `json:"unitInCar"`
				ClimatizationAtUnlock bool      `json:"climatizationAtUnlock"` // ClimatizationAtUnlock?
				WindowHeatingEnabled  bool      `json:"windowHeatingEnabled"`
				ZoneFrontLeftEnabled  bool      `json:"zoneFrontLeftEnabled"`
				ZoneFrontRightEnabled bool      `json:"zoneFrontRightEnabled"`
			} `json:"value"`
		} `json:"climatisationSettings"`
		WindowHeatingStatus struct {
			Value struct {
				CarCapturedTimestamp Timestamp `json:"carCapturedTimestamp"`
				WindowHeatingStatus  []struct {
					WindowLocation     string `json:"windowLocation"`
					WindowHeatingState string `json:"windowHeatingState"`
				} `json:"windowHeatingStatus"`
			} `json:"value"`
		} `json:"windowHeatingStatus"`
	} `json:"climatisation"`
	ClimatisationTimers *struct {
		ClimatisationTimersStatus struct {
			Value struct {
				Timers []struct {
					ID             int  `json:"id"`
					Enabled        bool `json:"enabled"`
					RecurringTimer struct {
						StartTime   string `json:"startTime"`
						RecurringOn struct {
							Mondays    bool `json:"mondays"`
							Tuesdays   bool `json:"tuesdays"`
							Wednesdays bool `json:"wednesdays"`
							Thursdays  bool `json:"thursdays"`
							Fridays    bool `json:"fridays"`
							Saturdays  bool `json:"saturdays"`
							Sundays    bool `json:"sundays"`
						} `json:"recurringOn"`
					} `json:"recurringTimer,omitempty"`
					SingleTimer struct {
						StartDateTime Timestamp `json:"startDateTime"`
					} `json:"singleTimer,omitempty"`
				} `json:"timers"`
				CarCapturedTimestamp Timestamp `json:"carCapturedTimestamp"`
				TimeInCar            Timestamp `json:"timeInCar"`
			} `json:"value"`
		} `json:"climatisationTimersStatus"`
	} `json:"climatisationTimers"`
	FuelStatus *struct {
		RangeStatus struct {
			Value struct {
				CarCapturedTimestamp Timestamp `json:"carCapturedTimestamp"`
				CarType              string    `json:"carType"`
				PrimaryEngine        struct {
					Type             string `json:"type"`
					CurrentSOCPct    int    `json:"currentSOC_pct"`
					RemainingRangeKm int    `json:"remainingRange_km"`
				} `json:"primaryEngine"`
				TotalRangeKm int `json:"totalRange_km"`
			} `json:"value"`
		} `json:"rangeStatus"`
	} `json:"fuelStatus"`
	Readiness *struct {
		ReadinessStatus struct {
			Value struct {
				ConnectionState struct {
					IsOnline                  bool   `json:"isOnline"`
					IsActive                  bool   `json:"isActive"`
					BatteryPowerLevel         string `json:"batteryPowerLevel"`
					DailyPowerBudgetAvailable bool   `json:"dailyPowerBudgetAvailable"`
				} `json:"connectionState"`
				ConnectionWarning struct {
					InsufficientBatteryLevelWarning bool `json:"insufficientBatteryLevelWarning"`
					DailyPowerBudgetWarning         bool `json:"dailyPowerBudgetWarning"`
				} `json:"connectionWarning"`
			} `json:"value"`
		} `json:"readinessStatus"`
	} `json:"readiness"`
	ChargingProfiles *struct {
		ChargingProfilesStatus struct {
			Value struct {
				CarCapturedTimestamp Timestamp     `json:"carCapturedTimestamp"`
				TimeInCar            Timestamp     `json:"timeInCar"`
				Profiles             []interface{} `json:"profiles"`
			} `json:"value"`
		} `json:"chargingProfilesStatus"`
	} `json:"chargingProfiles"`
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
