package keys

const (
	// loadpoint settings
	Title            = "title"       // loadpoint title
	Mode             = "mode"        // charge mode
	DefaultMode      = "defaultMode" // default charge mode
	Charger          = "charger"     // charger ref
	Meter            = "meter"       // meter ref
	Circuit          = "circuit"     // circuit ref
	DefaultVehicle   = "vehicle"     // default vehicle ref
	Priority         = "priority"    // priority
	MinCurrent       = "minCurrent"  // min current
	MaxCurrent       = "maxCurrent"  // max current
	MinSoc           = "minSoc"      // min soc
	LimitSoc         = "limitSoc"    // limit soc
	LimitEnergy      = "limitEnergy" // limit energy
	Soc              = "soc"
	Thresholds       = "thresholds"
	EnableThreshold  = "enableThreshold"
	DisableThreshold = "disableThreshold"
	EnableDelay      = "enableDelay"
	DisableDelay     = "disableDelay"
	BatteryBoost     = "batteryBoost"

	PhasesConfigured = "phasesConfigured" // desired phase mode (0/1/3, 0 = automatic), user selection
	PhasesActive     = "phasesActive"     // active phases as used by vehicle (1/2/3)

	ChargerIcon         = "chargerIcon"         // charger icon for ui
	ChargerFeature      = "chargerFeature"      // charger feature
	ChargerSinglePhase  = "chargerSinglePhase"  // api.PhaseDescriber: charger physical phases, sockets only
	ChargerPhases1p3p   = "chargerPhases1p3p"   // api.PhaseSwitcher: 1p3p chargers
	ChargerStatusReason = "chargerStatusReason" // either awaiting authorization or disconnect required

	// loadpoint status
	Enabled   = "enabled"   // loadpoint enabled
	Connected = "connected" // connected
	Charging  = "charging"  // charging

	// loadpoint setpoint
	OfferedCurrent = "offeredCurrent" // offered current

	// smart charging
	SmartCostActive    = "smartCostActive"    // smart cost active
	SmartCostLimit     = "smartCostLimit"     // smart cost limit
	SmartCostNextStart = "smartCostNextStart" // smart cost next start

	// effective values
	EffectivePriority   = "effectivePriority"   // effective priority
	EffectivePlanId     = "effectivePlanId"     // effective plan id
	EffectivePlanTime   = "effectivePlanTime"   // effective plan time
	EffectivePlanSoc    = "effectivePlanSoc"    // effective plan soc
	EffectiveMinCurrent = "effectiveMinCurrent" // effective min current
	EffectiveMaxCurrent = "effectiveMaxCurrent" // effective max current
	EffectiveLimitSoc   = "effectiveLimitSoc"   // effective limit soc

	// measurements
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
	PlanPrecondition   = "planPrecondition"   // charge plan precondition duration
	PlanActive         = "planActive"         // charge plan has determined current slot to be an active slot
	PlanProjectedStart = "planProjectedStart" // charge plan start time (earliest slot)
	PlanProjectedEnd   = "planProjectedEnd"   // charge plan ends (end of last slot)
	PlanOverrun        = "planOverrun"        // charge plan goal not reachable in time

	// repeating plans
	RepeatingPlans = "repeatingPlans" // key to access all repeating plans in db

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
	VehicleLimitSoc        = "vehicleLimitSoc"        // vehicle api soc limit
	VehicleClimaterActive  = "vehicleClimaterActive"  // vehicle climater active
	VehicleWelcomeActive   = "vehicleWelcomeActive"   // vehicle might need welcome charge
)
