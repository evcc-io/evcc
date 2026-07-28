package eebus

// EEBUS use case scenario numbers per the respective Use Case Technical Specifications.
// Each block mirrors the scenarios registered in the corresponding eebus-go usecase.

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
