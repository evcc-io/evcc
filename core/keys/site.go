package keys

const (
	Aux                   = "aux"
	AuxPower              = "auxPower"
	Circuits              = "circuits"
	Currency              = "currency"
	Ext                   = "ext"
	GreenShareHome        = "greenShareHome"
	GreenShareLoadpoints  = "greenShareLoadpoints"
	GridConfigured        = "gridConfigured"
	Grid                  = "grid"
	HomePower             = "homePower"
	PrioritySoc           = "prioritySoc"
	Pv                    = "pv"
	PvEnergy              = "pvEnergy"
	PvPower               = "pvPower"
	ResidualPower         = "residualPower"
	SiteTitle             = "siteTitle"
	SmartCostType         = "smartCostType"
	Statistics            = "statistics"
	Forecast              = "forecast"
	SolarAccYield         = "solarAccYield"
	SolarAccForecast      = "solarAccForecast"
	TariffCo2             = "tariffCo2"
	TariffCo2Home         = "tariffCo2Home"
	TariffCo2Loadpoints   = "tariffCo2Loadpoints"
	TariffFeedIn          = "tariffFeedIn"
	TariffGrid            = "tariffGrid"
	TariffPriceHome       = "tariffPriceHome"
	TariffPriceLoadpoints = "tariffPriceLoadpoints"
	TariffSolar           = "tariffSolar"
	Vehicles              = "vehicles"

	// meters
	GridMeter     = "gridMeter"
	PvMeters      = "pvMeters"
	BatteryMeters = "batteryMeters"
	ExtMeters     = "extMeters"
	AuxMeters     = "auxMeters"

	// battery settings
	BatteryCapacity         = "batteryCapacity"
	BatteryDischargeControl = "batteryDischargeControl"
	BatteryGridChargeLimit  = "batteryGridChargeLimit"
	BatteryGridChargeActive = "batteryGridChargeActive"
	BufferSoc               = "bufferSoc"
	BufferStartSoc          = "bufferStartSoc"

	// battery status
	Battery       = "battery"
	BatteryEnergy = "batteryEnergy"
	BatteryMode   = "batteryMode"
	BatteryPower  = "batteryPower"
	BatterySoc    = "batterySoc"

	// external battery control
	BatteryModeExternal = "batteryModeExternal"

	// smart charging
	SmartCostAvailable = "smartCostAvailable" // smart cost available

	// smart feed-in
	SmartFeedinDisableLimit     = "smartFeedinDisableLimit"     // smart feed-in disable limit
	SmartFeedinDisableActive    = "smartFeedinDisableActive"    // smart feed-in disable active
	SmartFeedinDisableAvailable = "smartFeedinDisableAvailable" // smart feed-in disable available

	SmartFeedinPriorityAvailable = "smartFeedinPriorityAvailable" // smart feed-in priority available
)
