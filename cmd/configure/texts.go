package configure

import "strings"

func (c *CmdConfigure) userFriendlyLabelHelp(label, help string) (string, string) {
	switch strings.ToLower(label) {
	case "title":
		label = "Titel"
		if help == "" {
			help = "Eine Text welcher in der Benutzeroberfläche angezeigt wird"
		}
	case "device":
		label = "Gerätadresse"
	case "baudrate":
		label = "Baudrate"
	case "comset":
		label = "ComSet"
	case "host":
		label = "IP Adresse oder den Namen"
	case "port":
		label = "Port Adresse"
	case "user":
		label = "Benutzername"
	case "password":
		label = "Passwort"
	case "capacity":
		label = "Akku-Kapazität in kWh"
	case "vin":
		label = "FIN"
		if help == "" {
			help = "FIN (Fahrzeugidentifikationsnummer)"
		}
	case "identifier":
		label = "Identifikationsnummer"
		if help == "" {
			help = "Kann meist erst später eingetagen werden, siehe: https://docs.evcc.io/docs/guides/vehicles/#erkennung-des-fahrzeugs-an-der-wallbox"
		}
	case "standbypower":
		label = "Standby-Leistung in W"
		if help == "" {
			help = "Leistung oberhalb des angegebenen Wertes, wird als Ladeleistung gewertet"
		}
	}
	return label, help
}
