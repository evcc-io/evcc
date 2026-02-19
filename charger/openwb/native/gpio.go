package native

/*
GPIO pin assignments for OpenWB hardware:
*/

type ChargePointGPIO struct {
	// CP-Unterbrechung (NC) und Freigabe Phasenumschaltung (NO)
	PIN_CP int
	// 1 phasig, Schütz B (L2+L3) gesperrt, bistabiles Relais (A)
	PIN_1P int
	// 3 phasig, Schütz B (L2+L3) freigegeben, bistabiles Relais (B)
	PIN_3P int
}

var ChargePoints = [2]ChargePointGPIO{
	// Chargepoint 0
	{
		PIN_CP: 25,
		PIN_1P: 5,
		PIN_3P: 26,
	},
	// Chargepoint 1
	{
		PIN_CP: 22,
		PIN_1P: 17,
		PIN_3P: 27,
	},
}
