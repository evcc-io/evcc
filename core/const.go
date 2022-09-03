package core

const (
	phasesConfigured = "phasesConfigured" // configured phases (1/3, 0 for auto on 1p3p chargers, nil for plain chargers)
	phasesEnabled    = "phasesEnabled"    // enabled phases (1/3)
	phasesActive     = "phasesActive"     // active phases as used by vehicle (1/2/3)

	vehicleDetectionActive = "vehicleDetectionActive" // vehicle detection is active (bool)

	vehicleRange     = "vehicleRange"     // vehicle range
	vehicleOdometer  = "vehicleOdometer"  // vehicle odometer
	vehicleTargetSoC = "vehicleTargetSoC" // vehicle soc limit
)
