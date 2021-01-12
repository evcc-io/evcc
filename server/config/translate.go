package config

var translations = map[string]string{
	"Cache":    "Aktualisierungsintervall",
	"Capacity": "Batteriekapazität (kWh)",
	"Password": "Passwort",
	"User":     "Username",
	"Title":    "Titel",
}

func translate(v string) string {
	return translations[v]
}
