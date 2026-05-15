package eebus

// EEBUS use case scenario numbers per the respective Use Case Technical Specifications.
//
// Spec scenario numbers diverge between use cases (e.g. MPC scenario 1 = active power,
// MGCP scenario 1 = power factor; MPC scenario 2 = energy, MGCP scenario 2 = active power).
// Passing the wrong number to IsScenarioAvailableAtEntity gates reads on the wrong feature.
//
// Each block mirrors the scenarios registered in the corresponding eebus-go usecase, which
// in turn matches the EEBus UC TS document.

// MGCP — Monitoring of Grid Connection Point (UC TS v1.0.0)
const (
	MGCPPowerFactor     uint = 1 // S1 power factor (cos phi)
	MGCPPower           uint = 2 // S2 active power per phase + total
	MGCPEnergyFeedIn    uint = 3 // S3 total feed-in energy
	MGCPEnergyConsumed  uint = 4 // S4 total consumed energy
	MGCPCurrentPerPhase uint = 5 // S5 phase-specific currents
	MGCPVoltagePerPhase uint = 6 // S6 phase-specific voltages
	MGCPFrequency       uint = 7 // S7 frequency
)

// MPC — Monitoring of Power Consumption (UC TS v1.0.0)
const (
	MPCPower           uint = 1 // S1 active power per phase + total
	MPCEnergyConsumed  uint = 2 // S2 total consumed energy
	MPCCurrentPerPhase uint = 3 // S3 phase-specific currents
	MPCVoltagePerPhase uint = 4 // S4 phase-specific voltages
	MPCFrequency       uint = 5 // S5 frequency
)

// LPC — Limitation of Power Consumption (UC TS v1.0.0). Same scenario layout for CS and EG roles.
const (
	LPCLimit                uint = 1 // S1 LoadControl: consumption limit
	LPCFailsafe             uint = 2 // S2 DeviceConfiguration: failsafe values
	LPCHeartbeat            uint = 3 // S3 DeviceDiagnosis: heartbeat
	LPCElectricalConnection uint = 4 // S4 ElectricalConnection (optional)
)

// LPP — Limitation of Power Production (UC TS v1.0.0). Same scenario layout for CS and EG roles.
const (
	LPPLimit                uint = 1 // S1 LoadControl: production limit
	LPPFailsafe             uint = 2 // S2 DeviceConfiguration: failsafe values
	LPPHeartbeat            uint = 3 // S3 DeviceDiagnosis: heartbeat
	LPPElectricalConnection uint = 4 // S4 ElectricalConnection (optional)
)

// OPEV — Overload Protection by EV Charging Current Curtailment (UC TS v1.0.1)
const (
	OPEVObligationLimit uint = 1 // S1 LoadControl + ElectricalConnection
	OPEVChargingState   uint = 2 // S2 charging state
	OPEVChargingPlan    uint = 3 // S3 charging plan
)

// OSCEV — Optimization of Self-Consumption during EV Charging (UC TS v1.0.1)
const (
	OSCEVRecommendationLimit uint = 1 // S1 LoadControl + ElectricalConnection
	OSCEVChargingState       uint = 2 // S2 charging state
	OSCEVChargingPlan        uint = 3 // S3 charging plan
)

// EVCEM — Measurement of Electricity during EV Charging (UC TS v1.0.1)
const (
	EVCEMPowerPerPhase uint = 1 // S1 phase-specific active power + ElectricalConnection (currents)
	EVCEMPowerTotal    uint = 2 // S2 total active power only
	EVCEMEnergy        uint = 3 // S3 charging energy summary
)

// EVSOC — EV State of Charge (UC TS v1.0.0 RC1)
const (
	EVSOCStateOfCharge uint = 1 // S1 state of charge
)
