package config

var translations = map[string]string{
	"Broker":   "Broker Adresse",
	"Cache":    "Aktualisierungsintervall",
	"Capacity": "Batteriekapazität (kWh)",
	"Password": "Passwort",
	"User":     "Username",
	"Scale":    "Multiplikator",
	"Serial":   "Seriennummer",
	"Title":    "Titel",
	"Power":    "Leistung (W)",
	"Energy":   "Energie (kWh)",
	"SoC":      "Ladezustand (%)",
	"Currents": "Phasenstrom (A)",
}

func translate(v string) string {
	return translations[v]
}
