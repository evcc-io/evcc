package core

const (
	phasesConfigured = "phasesConfigured" // configured phases (1/3, 0 for auto on 1p3p chargers, nil for plain chargers)
	phasesEnabled    = "phasesEnabled"    // enabled phases (1/3)
	phasesActive     = "phasesActive"     // active phases as used by vehicle (1/2/3)

	vehicleDetectionActive = "vehicleDetectionActive" // vehicle detection is active (bool)

	vehicleOdometer  = "vehicleOdometer"  // vehicle odometer
	vehicleRange     = "vehicleRange"     // vehicle range
	vehicleSoc       = "vehicleSoc"       // vehicle soc
	vehicleTargetSoc = "vehicleTargetSoc" // vehicle soc limit
	vehicleCapacity  = "vehicleCapacity"  // vehicle battery capacity
	vehicleIcon      = "vehicleIcon"      // vehicle icon for ui
	vehiclePresent   = "vehiclePresent"   // vehicle detected
	vehicleTitle     = "vehicleTitle"     // vehicle title

	minSoc                   = "minSoc"                   // min soc goal
	targetSoc                = "targetSoc"                // target charging soc goal
	targetTime               = "targetTime"               // target charging finish time goal
	targetTimeActive         = "targetTimeActive"         // target charging plan has determined current slot to be an active slot
	targetTimeProjectedStart = "targetTimeProjectedStart" // target charging plan start time (earliest slot)
)
