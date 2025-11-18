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
