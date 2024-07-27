package cupra

type Vehicle struct {
	VIN              string
	EnrollmentStatus string
	UserRole         string
	VehicleNickname  string
}

type Engine struct {
	Type     string
	FuelType string
	RangeKm  float64
	LevelPct float64
}

//	{
//	    "engines": {
//	        "primary": {
//	            "fuelType": "electric",
//	            "rangeKm": 298.0,
//	            "levelPct": 80.0
//	        }
//	    },
//	    "services": {
//	        "charging": {
//	            "status": "ChargePurposeReachedAndNotConservationCharging",
//	            "targetPct": 80,
//	            "currentPct": 80,
//	            "chargeSettings": "default",
//	            "chargeMode": "manual",
//	            "preferredChargeMode": "manual",
//	            "active": false,
//	            "remainingTime": 0,
//	            "progressBarPct": 100.0
//	        },
//	        "climatisation": {
//	            "status": "Off",
//	            "targetTemperatureCelsius": 20.0,
//	            "targetTemperatureFahrenheit": 68.0,
//	            "climatisationTrigger": "off",
//	            "active": false,
//	            "remainingTime": 0,
//	            "progressBarPct": 0.0
//	        },
//	        "windowHeating": {
//	            "status": "Off",
//	            "active": false
//	        }
//	    }
//	}
type Status struct {
	Engines struct {
		Primary, Secondary Engine
	}
	Services struct {
		Charging struct {
			Status         string
			TargetPct      int
			ChargeMode     string
			Active         bool
			RemainingTime  int64
			CurrentPct     float64
			ProgressBarPct float64
		}
		Climatisation struct {
			Status         string
			Active         bool
			RemainingTime  int64
			ProgressBarPct float64
		}
	}
	Measurements struct {
		MileageKm float64
	}
}

type Mileage struct {
	MileageKm float64
}

type Position struct {
	Lat, Lon float64
}
