package hw

/*
GPIO pin assignments for OpenWB hardware:

Chargepoint 0:
GPIO 25 => CP-Unterbrechung (NC) und Freigabe Phasenumschaltung (NO)
GPIO  5 => 1 phasig, Schütz B (L2+L3) gesperrt, bistabiles Relais (A)
GPIO 26 => 3 phasig, Schütz B (L2+L3) freigegeben, bistabiles Relais (B)

Chargepoint 1:
GPIO 22 => CP-Unterbrechung (NC) und Freigabe Phasenumschaltung (NO)
GPIO 17 => 1 phasig, Schütz B (L2+L3) gesperrt, bistabiles Relais (A)
GPIO 27 => 3 phasig, Schütz B (L2+L3) freigegeben, bistabiles Relais (B)

*/

var (
	GPIO_CP = [2]int{25, 22}
	GPIO_1P = [2]int{5, 17}
	GPIO_3P = [2]int{26, 27}
)
