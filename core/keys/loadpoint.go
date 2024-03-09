package keys

const (
	// loadpoint settings
	Title            = "title"       // loadpoint title
	Mode             = "mode"        // charge mode
	Priority         = "priority"    // priority
	MinCurrent       = "minCurrent"  // min current
	MaxCurrent       = "maxCurrent"  // max current
	MinSoc           = "minSoc"      // min soc
	LimitSoc         = "limitSoc"    // limit soc
	LimitEnergy      = "limitEnergy" // limit energy
	EnableThreshold  = "enableThreshold"
	DisableThreshold = "disableThreshold"

	PhasesConfigured = "phasesConfigured" // configured phases (1/3, 0 for auto on 1p3p chargers, nil for plain chargers)
	PhasesEnabled    = "phasesEnabled"    // enabled phases (1/3)
	PhasesActive     = "phasesActive"     // active phases as used by vehicle (1/2/3)

	ChargerIcon           = "chargerIcon"           // charger icon for ui
	ChargerFeature        = "chargerFeature"        // charger feature
	ChargerPhysicalPhases = "chargerPhysicalPhases" // charger phases
	ChargerPhases1p3p     = "chargerPhases1p3p"     // phase switcher (1p3p chargers)

	// loadpoint status
	Enabled   = "enabled"   // loadpoint enabled
	Connected = "connected" // connected
	Charging  = "charging"  // charging

	// smart charging
	SmartCostActive = "smartCostActive" // smart cost active
	SmartCostLimit  = "smartCostLimit"  // smart cost limit

	// effective values
	EffectivePriority   = "effectivePriority"   // effective priority
	EffectivePlanTime   = "effectivePlanTime"   // effective plan time
	EffectivePlanSoc    = "effectivePlanSoc"    // effective plan soc
	EffectiveMinCurrent = "effectiveMinCurrent" // effective min current
	EffectiveMaxCurrent = "effectiveMaxCurrent" // effective max current
	EffectiveLimitSoc   = "effectiveLimitSoc"   // effective limit soc

	// measurements
	ChargeCurrent     = "chargeCurrent"     // charge current
	ChargePower       = "chargePower"       // charge power
	ChargeCurrents    = "chargeCurrents"    // charge currents
	ChargeVoltages    = "chargeVoltages"    // charge voltages
	ChargedEnergy     = "chargedEnergy"     // charged energy
	ChargeDuration    = "chargeDuration"    // charge duration
	ChargeTotalImport = "chargeTotalImport" // charge meter total import

	// session
	ConnectedDuration       = "connectedDuration"       // connected duration
	ChargeRemainingDuration = "chargeRemainingDuration" // charge remaining duration
	ChargeRemainingEnergy   = "chargeRemainingEnergy"   // charge remaining energy

	// plan
	PlanTime           = "planTime"           // charge plan finish time goal
	PlanEnergy         = "planEnergy"         // charge plan energy goal
	PlanSoc            = "planSoc"            // charge plan soc goal
	PlanActive         = "planActive"         // charge plan has determined current slot to be an active slot
	PlanProjectedStart = "planProjectedStart" // charge plan start time (earliest slot)
	PlanOverrun        = "planOverrun"        // charge plan goal not reachable in time

	// remote control
	RemoteDisabled       = "remoteDisabled"       // remote disabled
	RemoteDisabledSource = "remoteDisabledSource" // remote disabled source

	// vehicle
	VehicleName            = "vehicleName"            // vehicle name
	VehicleIdentity        = "vehicleIdentity"        // vehicle identity
	VehicleDetectionActive = "vehicleDetectionActive" // vehicle detection active
	VehicleOdometer        = "vehicleOdometer"        // vehicle odometer
	VehicleRange           = "vehicleRange"           // vehicle range
	VehicleSoc             = "vehicleSoc"             // vehicle soc
	VehicleTargetSoc       = "vehicleTargetSoc"       // vehicle api soc limit
	VehicleClimaterActive  = "vehicleClimaterActive"  // vehicle climater active
)
