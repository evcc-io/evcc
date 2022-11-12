package telemetry

type InstanceChargeProgress struct {
	InstanceID string `json:"instanceId"`
	ChargeProgress
}

type ChargeProgress struct {
	ChargePower  float64 `json:"chargePower"`
	GreenPower   float64 `json:"greenPower"`
	ChargeEnergy float64 `json:"chargeEnergy"`
	GreenEnergy  float64 `json:"greenEnergy"`
}
