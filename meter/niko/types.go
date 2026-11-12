package niko

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	TopicCmd = "hobby/control/devices/cmd"
	TopicRsp = "hobby/control/devices/rsp"
	TopicEvt = "hobby/control/devices/evt"

	MethodList    = "devices.list"
	MethodControl = "devices.control"

	TypeSmartPlug = "smartplug"
)

// Message is a Hobby API request, response or event
type Message struct {
	Method string  `json:"Method"`
	Params []Param `json:"Params,omitempty"`
}

type Param struct {
	Devices []Device `json:"Devices,omitempty"`
}

// Device is a device as listed by devices.list or reported by devices.status
type Device struct {
	Uuid       string     `json:"Uuid"`
	Type       string     `json:"Type,omitempty"`
	Name       string     `json:"Name,omitempty"`
	Properties []Property `json:"Properties,omitempty"`
}

// Property is a single-key object, e.g. {"Status":"On"}
type Property map[string]any

// Devices returns all devices contained in the message
func (m Message) Devices() []Device {
	var res []Device
	for _, p := range m.Params {
		res = append(res, p.Devices...)
	}
	return res
}

// Property returns the string value of the given property
func (d Device) Property(key string) (string, bool) {
	for _, p := range d.Properties {
		if v, ok := p[key]; ok {
			return fmt.Sprint(v), true
		}
	}
	return "", false
}

// Status returns the relay state
func (d Device) Status() (bool, bool) {
	v, ok := d.Property("Status")
	return strings.EqualFold(v, "On"), ok
}

// Power returns the instantaneous power in W
func (d Device) Power() (float64, bool) {
	v, ok := d.Property("ElectricalPower")
	if !ok {
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// ListRequest creates a devices.list request
func ListRequest() string {
	b, _ := json.Marshal(Message{Method: MethodList})
	return string(b)
}

// ControlRequest creates a devices.control request setting a single property
func ControlRequest(uuid, key, value string) string {
	b, _ := json.Marshal(Message{
		Method: MethodControl,
		Params: []Param{{Devices: []Device{{
			Uuid:       uuid,
			Properties: []Property{{key: value}},
		}}}},
	})
	return string(b)
}
