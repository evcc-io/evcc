package porsche

import (
	"errors"

	"github.com/evcc-io/evcc/api"
)

type Vehicle struct {
	VIN              string
	ModelDescription string
	Pictures         []struct {
		URL         string
		View        string
		Size        int
		Width       int
		Height      int
		Transparent bool
	}
}

type VehiclePairingResponse struct {
	VIN                string
	PairingCode        string
	Status             string
	CanSendPairingCode bool
}

type MeasurementMobile struct {
	Key string
	/*
		RANGE,
		E_RANGE,
		MILEAGE,
		FUEL_LEVEL,
		FUEL_RESERVE,
		BATTERY_LEVEL // status, value
		BATTERY_CHARGING_STATE // status, value
		OIL_SERVICE_RANGE,
		OIL_SERVICE_TIME,
		MAIN_SERVICE_RANGE,
		MAIN_SERVICE_TIME,
		INTERMEDIATE_SERVICE_RANGE,
		INTERMEDIATE_SERVICE_TIME,
		OIL_LEVEL_MAX,
		OIL_LEVEL_CURRENT,
		OIL_LEVEL_MIN_WARNING,
		OPEN_STATE_DOOR_FRONT_LEFT,
		OPEN_STATE_DOOR_REAR_LEFT,
		OPEN_STATE_DOOR_FRONT_RIGHT,
		OPEN_STATE_DOOR_REAR_RIGHT,
		OPEN_STATE_LID_FRONT,
		OPEN_STATE_LID_REAR,
		OPEN_STATE_WINDOW_FRONT_LEFT,
		OPEN_STATE_WINDOW_REAR_LEFT,
		OPEN_STATE_WINDOW_FRONT_RIGHT,
		OPEN_STATE_WINDOW_REAR_RIGHT,
		OPEN_STATE_SUNROOF,
		OPEN_STATE_SUNROOF_REAR,
		OPEN_STATE_TOP,
		OPEN_STATE_SERVICE_FLAP,
		OPEN_STATE_SPOILER,
		OPEN_STATE_CHARGE_FLAP_LEFT,
		OPEN_STATE_CHARGE_FLAP_RIGHT,
		LOCK_STATE_VEHICLE,
		GPS_LOCATION,
		GLOBAL_PRIVACY_MODE,
		REMOTE_ACCESS_AUTHORIZATION,
		THEFT_MODE,
		TRIP_STATISTICS_CYCLIC,
		TRIP_STATISTICS_LONG_TERM,
		TRIP_STATISTICS_SHORT_TERM,
		PARKING_LIGHT,
		HEATING_STATE,
		ACV_STATE,
		CLIMATIZER_STATE // isOn, minutesLeft, targetTemperature
		TIMERS,
		BLEID_DDADATA,
		VTS_MODES,
		SPEED_ALARMS,
		LOCATION_ALARMS,
		CHARGING_PROFILES,
		BATTERY_TYPE // lastModified, plugTypes, capacityAh capacityKWh
		VALET_ALARM;
	*/
	Status struct {
		IsEnabled bool
		Cause     string // AVAILABLE, PRIVACY_ACTIVATED, LICENSE_DEACTIVATED, NOT_SUPPORTED, UNKNOWN
	}
	Value struct {
		LastModified string

		IsEnabled              bool
		SocPhoneNumber         string
		Percent                int64
		DistanceKM             float64
		ZeroEmissionDistanceKm float64
		AvgLiterPerHundredKm   float64
		AvgKwhPerHundredKm     float64
		AvgSpeedKmh            float64
		DrivingTimeMinutes     float64
		TripEndTime            string

		IsOn              bool
		TargetTemperature float64
		// MinutesLeft

		Kilometers int64
		Days       int64
		IsLocked   bool
		IsOpen     bool
		Direction  int64

		PlugTypes string // COMBINED_CS, TYP2, UNKNOWN
		// CapacityAh
		// CapacityKWh

		Status              string // CHARGING, FAST_CHARGING, CHARGING_COMPLETED, CHARGING_PAUSED, READY_TO_CHARGE, SOC_REACHED, INITIALISING, STANDBY, SUSPENDED, PLUGGED_LOCKED, PLUGGED_NOT_LOCKED, CHARGING_ERROR, NOT_PLUGGED, UNKNOWN
		Mode                string // DIRECT, TIMER_PROFILE, LONGTERM, TCP, UNKNOWN
		DirectChargingState string // ENABLED_ON, ENABLED_OFF, DISABLED_ON, DISABLED_OFF, HIDDEN, UNKNOWN
		EndsAt              string
		ChargingPower       float64
		ChargingRate        float64
	}
}

type StatusResponseMobile struct {
	VIN        string
	ModelName  string
	CustomName string
	ModelType  struct {
		Code       string
		Year       string
		Body       string
		Generation string
		Model      string
		Engine     string
	}
	Connect             bool
	GreyConnectStoreURL string
	PairingCodeV2       string
	Measurements        []MeasurementMobile
}

func (s *StatusResponseMobile) MeasurementByKey(key string) (*MeasurementMobile, error) {
	for _, m := range s.Measurements {
		if m.Key == key {
			var err error
			if !m.Status.IsEnabled {
				switch m.Status.Cause {
				case "UNKNOWN":
					break
				case "PRIVACY_ACTIVATED", "LICENSE_DEACTIVATED", "NOT_SUPPORTED":
					err = api.ErrNotAvailable
				default:
					err = errors.New(m.Status.Cause)
				}
			}
			return &m, err
		}
	}
	return nil, api.ErrNotAvailable
}

type StatusResponse struct {
	BatteryLevel struct {
		Unit  string
		Value float64
	}
	Mileage struct {
		Unit  string
		Value float64
	}
	RemainingRanges struct {
		ElectricalRange struct {
			Distance struct {
				Unit  string
				Value float64
			}
		}
	}
}

type CapabilitiesResponse struct {
	DisplayParkingBrake      bool
	NeedsSPIN                bool
	HasRDK                   bool
	EngineType               string
	CarModel                 string
	OnlineRemoteUpdateStatus struct {
		EditableByUser bool
		Active         bool
	}
	HeatingCapabilities struct {
		FrontSeatHeatingAvailable bool
		RearSeatHeatingAvailable  bool
	}
	SteeringWheelPosition string
	HasHonkAndFlash       bool
}

type EmobilityResponse struct {
	BatteryChargeStatus *struct {
		ChargeRate struct {
			Unit             string
			Value            float64
			ValueInKmPerHour int64
		}
		ChargingInDCMode                            bool
		ChargingMode                                string
		ChargingPower                               float64
		ChargingReason                              string
		ChargingState                               string
		ChargingTargetDateTime                      string
		ExternalPowerSupplyState                    string
		PlugState                                   string
		RemainingChargeTimeUntil100PercentInMinutes int64
		StateOfChargeInPercentage                   int64
		RemainingERange                             struct {
			OriginalUnit      string
			OriginalValue     int64
			Unit              string
			Value             int64
			ValueInKilometers int64
		}
	}
	ChargingStatus string
	DirectCharge   struct {
		Disabled bool
		IsActive bool
	}
	DirectClimatisation struct {
		ClimatisationState         string
		RemainingClimatisationTime int64
	}
	PcckErrorMessage string
}
