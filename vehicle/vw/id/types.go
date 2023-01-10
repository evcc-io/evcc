package id

import (
	"errors"
	"fmt"
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
				ChargePowerKW                      float64   `json:"chargePower_kW"`
				ChargeRateKmph                     float64   `json:"chargeRate_kmph"`
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
	FuelStatus *FuelStatus `json:"fuelStatus"`
	Readiness  *struct {
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

// FuelStatus is the engine range status
type FuelStatus struct {
	RangeStatus struct {
		Value struct {
			CarCapturedTimestamp Timestamp         `json:"carCapturedTimestamp"`
			CarType              string            `json:"carType"`
			PrimaryEngine        EngineRangeStatus `json:"primaryEngine"`
			SecondaryEngine      EngineRangeStatus `json:"secondaryEngine"`
			TotalRangeKm         int               `json:"totalRange_km"`
		} `json:"value"`
	} `json:"rangeStatus"`
}

func (f *FuelStatus) EngineRangeStatus(typ string) (EngineRangeStatus, error) {
	if f == nil {
		return EngineRangeStatus{}, errors.New("missing fuel status")
	}

	if f.RangeStatus.Value.PrimaryEngine.Type == typ {
		return f.RangeStatus.Value.PrimaryEngine, nil
	}
	if f.RangeStatus.Value.SecondaryEngine.Type == typ {
		return f.RangeStatus.Value.SecondaryEngine, nil
	}

	return EngineRangeStatus{}, fmt.Errorf("unknown engine type: %s", typ)
}

// EngineRangeStatus is the engine range status
type EngineRangeStatus struct {
	Type             string `json:"type"`
	CurrentSOCPct    int    `json:"currentSOC_pct"`
	RemainingRangeKm int    `json:"remainingRange_km"`
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
