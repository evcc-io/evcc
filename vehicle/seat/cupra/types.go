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
	Range    struct {
		Value float64
		Unit  string
	}
	Level float64
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
			Active         bool
			RemainingTime  int64
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
