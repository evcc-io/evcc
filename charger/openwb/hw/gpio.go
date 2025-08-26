package hw

/*
GPIO pin assignments for OpenWB hardware:

GPIO 25 => CP-Unterbrechung (NC) und Freigabe Phasenumschaltung (NO)
GPIO  5 => 1 phasig, Schütz B (L2+L3) gesperrt, bistabiles Relais (A)
GPIO 26 => 3 phasig, Schütz B (L2+L3) freigegeben, bistabiles Relais (B)
*/

const (
	GPIO_CP = 25
	GPIO_1P = 5
	GPIO_3P = 26
)
