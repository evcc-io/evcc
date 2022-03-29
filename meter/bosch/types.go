package bosch

type LoginResponse struct {
	wuSid string
}

type StatusResponse struct {
	currentBatterySoc     float64
	sellToGrid            float64
	buyFromGrid           float64
	pvPower               float64
	batteryChargePower    float64
	batteryDischargePower float64
}
