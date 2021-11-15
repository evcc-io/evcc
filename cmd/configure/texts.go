package configure

import (
	"strings"

	"github.com/evcc-io/evcc/templates"
	"github.com/thoas/go-funk"
)

func (c *CmdConfigure) userFriendlyTexts(param templates.Param) templates.Param {
	result := param

	if result.ValueType != "" && funk.ContainsString(templates.ParamValueTypes, result.ValueType) {
		result.ValueType = result.ValueType
	} else {
		result.ValueType = templates.ParamValueTypeString
	}

	switch strings.ToLower(result.Name) {
	case "title":
		result.Name = "Titel"
		if result.Help == "" {
			result.Help = "Eine Text welcher in der Benutzeroberfläche angezeigt wird"
		}
	case "device":
		result.Name = "Gerätadresse"
	case "baudrate":
		result.Name = "Baudrate"
	case "comset":
		result.Name = "ComSet"
	case "host":
		result.Name = "IP-Adresse oder Hostname"
	case "port":
		result.Name = "Port"
		result.ValueType = templates.ParamValueTypeNumber
	case "user":
		result.Name = "Benutzername"
	case "password":
		result.Name = "Passwort"
	case "capacity":
		result.Name = "Akku-Kapazität in kWh"
		if result.Example == "" {
			result.Example = "41.5"
		}
		result.ValueType = templates.ParamValueTypeFloat
	case "vin":
		result.Name = "Fahrzeugidentifikationsnummer"
		if result.Help == "" {
			result.Help = "Erforderlich, wenn mehrere Fahrzeuge des Herstellers vorhanden sind"
		}
	case "identifier":
		result.Name = "Identifikationsnummer"
		if result.Help == "" {
			result.Help = "Kann meist erst später eingetragen werden, siehe: https://docs.evcc.io/docs/guides/vehicles/#erkennung-des-fahrzeugs-an-der-wallbox"
		}
	case "standbypower":
		result.Name = "Standby-Leistung in W"
		if result.Help == "" {
			result.Help = "Leistung oberhalb des angegebenen Wertes, wird als Ladeleistung gewertet"
		}
		result.ValueType = templates.ParamValueTypeNumber
	}
	return result
}
