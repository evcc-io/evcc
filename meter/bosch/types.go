package bosch

type LoginResponse struct {
	wuSid string
}

type StatusResponse struct {
	CurrentBatterySoc     float64
	SellToGrid            float64
	BuyFromGrid           float64
	PvPower               float64
	BatteryChargePower    float64
	BatteryDischargePower float64
}
