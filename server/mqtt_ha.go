package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/core/site"
)

type HADeviceDef struct {
	Identifiers  string `json:"identifiers"`
	Manufacturer string `json:"manufacturer"`
	Name         string `json:"name"`
}

type HAEntityDef struct {
	Device   HADeviceDef `json:"device"`
	Name     string      `json:"name"`
	UniqueId string      `json:"unique_id"`

	StateTopic        string `json:"state_topic,omitempty"`
	CommandTopic      string `json:"command_topic,omitempty"`
	AvailabilityTopic string `json:"availability_topic,omitempty"`

	Options []string `json:"options,omitempty"`

	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

func (m *MQTT) haPublishBaseDeviceDef(site site.API, loadPoint *int, valueName string) HAEntityDef {
	siteId := strings.ReplaceAll(strings.ToLower(site.Name()), " ", "_") // TODO find better site ID
	uid := "evcc_" + siteId + "_"
	if loadPoint != nil {
		uid += fmt.Sprintf("lp%d_", *loadPoint+1)
	}
	uid += valueName

	var name string
	if loadPoint == nil {
		name = fmt.Sprintf("EVCC %s %s", site.Name(), valueName)
	} else {
		name = fmt.Sprintf("EVCC %s Loadpoint %d %s", site.Name(), *loadPoint+1, valueName)
	}

	return HAEntityDef{
		Device: HADeviceDef{
			Identifiers:  fmt.Sprintf("evcc_%s", siteId),
			Manufacturer: "EVCC",
			Name:         site.Name(),
		},
		Name:              name,
		UniqueId:          uid,
		AvailabilityTopic: m.root + "/status",
	}
}

func (m *MQTT) haPublishDiscoverSensors(site site.API, loadPoint *int, valueName, stateTopic string) {
	if _, ok := m.haKnownSensors[stateTopic]; ok {
		return
	}

	entityDef := m.haPublishBaseDeviceDef(site, loadPoint, valueName)
	entityDef.StateTopic = stateTopic
	topic := "homeassistant/sensor/" + entityDef.UniqueId + "/config"

	jsonData, _ := json.MarshalIndent(entityDef, "", "  ")
	token := m.Handler.Client.Publish(topic, m.Handler.Qos, true, jsonData)
	go m.Handler.WaitForToken(token)

	// mark as know to sensors publish only once
	m.haKnownSensors[stateTopic] = struct{}{}
}

func (m *MQTT) haPublishDiscoverSelect(site site.API, loadPoint *int, valueName, stateTopic string, options []string) {
	entityDef := m.haPublishBaseDeviceDef(site, loadPoint, valueName)

	entityDef.StateTopic = stateTopic
	entityDef.CommandTopic = stateTopic + "/set"
	entityDef.Options = options
	topic := "homeassistant/select/" + entityDef.UniqueId + "/config"

	jsonData, _ := json.MarshalIndent(entityDef, "", "  ")
	token := m.Handler.Client.Publish(topic, m.Handler.Qos, true, jsonData)
	go m.Handler.WaitForToken(token)

	// mark as know to sensors publish only once
	m.haKnownSensors[stateTopic] = struct{}{}
}

func (m *MQTT) haPublishDiscoverNumber(site site.API, loadPoint *int, valueName, stateTopic string, min, max float64) {
	entityDef := m.haPublishBaseDeviceDef(site, loadPoint, valueName)

	entityDef.StateTopic = stateTopic
	entityDef.CommandTopic = stateTopic + "/set"
	entityDef.Min = min
	entityDef.Max = max
	topic := "homeassistant/number/" + entityDef.UniqueId + "/config"

	jsonData, _ := json.MarshalIndent(entityDef, "", "  ")
	token := m.Handler.Client.Publish(topic, m.Handler.Qos, true, jsonData)
	go m.Handler.WaitForToken(token)

	// mark as know to sensors publish only once
	m.haKnownSensors[stateTopic] = struct{}{}
}
