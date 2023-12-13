package vw

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

type Error struct {
	ErrorCode, Description string
}

func (e *Error) Error() error {
	return fmt.Errorf("%s: %s", e.ErrorCode, e.Description)
}

// ChargerResponse is the /bs/batterycharge/v1/%s/%s/vehicles/%s/charger api
type ChargerResponse struct {
	Charger struct {
		Status struct {
			BatteryStatusData struct {
				StateOfCharge         TimedInt
				RemainingChargingTime TimedInt
			}
			ChargingStatusData struct {
				ChargingState            TimedString // off, charging
				ChargingMode             TimedString // invalid, AC
				ChargingReason           TimedString // invalid, immediate
				ExternalPowerSupplyState TimedString // unavailable, available
				EnergyFlow               TimedString // on, off
			}
			PlugStatusData struct {
				PlugState TimedString // connected
			}
			CruisingRangeStatusData struct {
				EngineTypeFirstEngine  TimedString // typeIsElectric, petrolGasoline
				EngineTypeSecondEngine TimedString // typeIsElectric, petrolGasoline
				PrimaryEngineRange     TimedInt
				SecondaryEngineRange   TimedInt
				HybridRange            TimedInt
			}
		}
	}
	Error *Error // optional error
}

// ClimaterResponse is the /bs/climatisation/v1/%s/%s/vehicles/%s/climater api
type ClimaterResponse struct {
	Climater struct {
		Settings struct {
			TargetTemperature TimedTemperature
			HeaterSource      TimedString
		}
		Status struct {
			ClimatisationStatusData struct {
				ClimatisationState         TimedString
				ClimatisationReason        TimedString
				RemainingClimatisationTime TimedInt
			}
			TemperatureStatusData struct {
				OutdoorTemperature TimedTemperature
			}
			VehicleParkingClockStatusData struct {
				VehicleParkingClock TimedString
			}
		}
	}
	Error *Error // optional error
}

// PositionResponse is the /bs/cf/v1/%s/%s/vehicles/%s/position api
type PositionResponse struct {
	FindCarResponse struct {
		Position struct {
			TimestampCarSent     string // "2021-12-12T16:42:44"
			TimestampTssReceived time.Time
			CarCoordinate        struct {
				Latitude  int64
				Longitude int64
			}
			TimestampCarSentUTC  time.Time
			TimestampCarCaptured string // "2021-12-12T16:42:44"
		}
		ParkingTimeUTC time.Time
	}
	Error *Error // optional error
}

// VehiclesResponse is the /usermanagement/users/v1/%s/%s/vehicles api
type VehiclesResponse struct {
	UserVehicles struct {
		Vehicle []string
	}
	Error *Error // optional error
}

// HomeRegion is the home region API response
type HomeRegion struct {
	HomeRegion struct {
		BaseURI struct {
			SystemID string
			Content  string // api url
		}
	}
	Error *Error // optional error
}

// TimedInt is an int value with timestamp
type TimedInt struct {
	Content   int
	Timestamp string
}

// TimedString is a string value with timestamp
type TimedString struct {
	Content   string
	Timestamp string
}

// TimedTemperature is the api temperature with timestamp
type TimedTemperature struct {
	Content   float64
	Timestamp string
}

func (t *TimedTemperature) UnmarshalJSON(data []byte) error {
	var temp struct {
		Content   json.RawMessage // handle "invalid"
		Timestamp string
	}

	err := json.Unmarshal(data, &temp)
	if err == nil {
		t.Timestamp = temp.Timestamp

		if val, err := strconv.Atoi(string(temp.Content)); err == nil {
			t.Content = temp2Float(val)
		} else {
			t.Content = math.NaN()
		}
	}

	return err
}

// temp2Float converts api temp to float value
func temp2Float(i int) float64 {
	f := float64(i)
	if f == 0 {
		return math.NaN()
	}
	return f/10 - 273
}
