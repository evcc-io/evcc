package cmd

import (
	"encoding/json"
	"errors"
)

type Class int

//go:generate enumer -type Class -trimprefix Class -transform=lower -text
const (
	_ Class = iota
	ClassConfigFile
	ClassMeter
	ClassCharger
	ClassVehicle
	ClassTariff
	ClassCircuit
	ClassSite
	ClassMqtt
	ClassDatabase
	ClassModbusProxy
	ClassEEBus
	ClassJavascript
	ClassGo
	ClassHEMS
	ClassInflux
	ClassMessenger
	ClassSponsorship
)

// FatalError is an error that can be marshaled
type FatalError struct {
	err error
}

func (e *FatalError) Error() string {
	return e.err.Error()
}

func (e FatalError) MarshalJSON() ([]byte, error) {
	if je, ok := e.err.(json.Marshaler); ok {
		return je.MarshalJSON()
	}

	return json.Marshal(struct {
		Error string `json:"error"`
	}{
		Error: e.err.Error(),
	})
}

// DeviceError indicates the specific device that failed
type DeviceError struct {
	Name string
	err  error
}

func (e *DeviceError) Error() string {
	return e.err.Error()
}

// ClassError indicates the class of devices that failed
type ClassError struct {
	Class Class
	err   error
}

func (e *ClassError) Error() string {
	return e.err.Error()
}

func (e ClassError) MarshalJSON() ([]byte, error) {
	res := struct {
		Class  string `json:"class"`
		Device string `json:"device,omitempty"`
		Error  string `json:"error"`
	}{
		Class: e.Class.String(),
		Error: e.err.Error(),
	}

	var de *DeviceError
	if errors.As(e.err, &de) {
		res.Device = de.Name
	}

	return json.Marshal(res)
}

func wrapErrorWithClass(class Class, err error) error {
	if err == nil {
		return nil
	}

	return &ClassError{
		Class: class,
		err:   err,
	}
}
