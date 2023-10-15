package core

const (
	title = "title" // loadpoint title

	phasesConfigured = "phasesConfigured" // configured phases (1/3, 0 for auto on 1p3p chargers, nil for plain chargers)
	phasesEnabled    = "phasesEnabled"    // enabled phases (1/3)
	phasesActive     = "phasesActive"     // active phases as used by vehicle (1/2/3)

	chargerIcon = "chargerIcon" // charger icon for ui

	vehicleCapacity        = "vehicleCapacity"        // vehicle battery capacity
	vehicleDetectionActive = "vehicleDetectionActive" // vehicle detection active
	vehicleIcon            = "vehicleIcon"            // vehicle icon for ui
	vehicleOdometer        = "vehicleOdometer"        // vehicle odometer
	vehiclePresent         = "vehiclePresent"         // vehicle detected
	vehicleRange           = "vehicleRange"           // vehicle range
	vehicleSoc             = "vehicleSoc"             // vehicle soc
	vehicleTargetSoc       = "vehicleTargetSoc"       // vehicle soc limit
	vehicleTitle           = "vehicleTitle"           // vehicle title

	minCurrent              = "minCurrent"              // charger min current
	maxCurrent              = "maxCurrent"              // charger max current
	chargeRemainingDuration = "chargeRemainingDuration" // charge remaining duration
	minSoc                  = "minSoc"                  // min soc
	limitSoc                = "limitSoc"                // limit soc

	planEnergy         = "planEnergy"         // charge plan energy goal
	planSoc            = "planSoc"            // charge plan soc goal
	planTime           = "planTime"           // charge plan finish time goal
	planActive         = "planActive"         // charge plan has determined current slot to be an active slot
	planProjectedStart = "planProjectedStart" // charge plan start time (earliest slot)
)
