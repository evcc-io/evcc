package config

type labelAndUnit struct{ label, unit string }

var translations = map[string]labelAndUnit{
	"Broker":   {label: "Broker Adresse"},
	"Cache":    {label: "Aktualisierungsintervall"},
	"Capacity": {label: "Batteriekapazität", unit: "kWh"},
	"Password": {label: "Passwort"},
	"User":     {label: "Username"},
	"Scale":    {label: "Multiplikator"},
	"Serial":   {label: "Seriennummer"},
	"Title":    {label: "Titel"},
	"VIN":      {label: "Fahrgestellnummer"},
	"Power":    {label: "Leistung", unit: "W"},
	"Energy":   {label: "Energie", unit: "kWh"},
	"SoC":      {label: "Ladezustand", unit: "%"},
	"Currents": {label: "Phasenstrom", unit: "A"},
}

func translate(v string) string {
	val, ok := translations[v]
	if ok {
		return val.label
	}
	return ""
}

func translateUnit(v string) string {
	val, ok := translations[v]
	if ok {
		return val.unit
	}
	return ""
}
