package config

var translations = map[string]string{
	"Broker":   "MQTT Broker Adresse",
	"Topic":    "MQTT Topic",
	"Cache":    "Aktualisierungsintervall",
	"Capacity": "Batteriekapazität (kWh)",
	"Password": "Passwort",
	"User":     "Username",
	"Serial":   "Seriennummer",
	"Title":    "Titel",
}

func translate(v string) string {
	return translations[v]
}
