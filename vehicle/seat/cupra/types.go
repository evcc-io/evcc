package cupra

const FuelTypeElectric = "electric"

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

type Status struct {
	Engines struct {
		Primary, Secondary Engine
	}
	Services struct {
		Charging struct {
			Status         string
			TargetPct      int
			ChargeMode     string
			ChargeSettings string
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
