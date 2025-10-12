package semp

import (
	"encoding/xml"
)

// Status constants
const (
	StatusOn  = "On"
	StatusOff = "Off"
)

// DeviceInfo represents SEMP device information
type DeviceInfo struct {
	Identification  Identification  `xml:"Identification"`
	Characteristics Characteristics `xml:"Characteristics"`
	Capabilities    Capabilities    `xml:"Capabilities"`
}

// Identification represents SEMP device identification
type Identification struct {
	DeviceID     string `xml:"DeviceId"`
	DeviceName   string `xml:"DeviceName"`
	DeviceType   string `xml:"DeviceType"`
	DeviceSerial string `xml:"DeviceSerial"`
	DeviceVendor string `xml:"DeviceVendor"`
}

// Characteristics represents SEMP device characteristics
type Characteristics struct {
	MinPowerConsumption int `xml:"MinPowerConsumption"`
	MaxPowerConsumption int `xml:"MaxPowerConsumption"`
	MinOnTime           int `xml:"MinOnTime,omitempty"`
	MinOffTime          int `xml:"MinOffTime,omitempty"`
}

// Capabilities represents SEMP device capabilities
type Capabilities struct {
	CurrentPowerMethod   string `xml:"CurrentPower>Method"`
	AbsoluteTimestamps   bool   `xml:"Timestamps>AbsoluteTimestamps"`
	InterruptionsAllowed bool   `xml:"Interruptions>InterruptionsAllowed"`
	OptionalEnergy       bool   `xml:"Requests>OptionalEnergy"`
}

// DeviceStatus represents SEMP device status
type DeviceStatus struct {
	DeviceID          string    `xml:"DeviceId"`
	Status            string    `xml:"Status"`
	EMSignalsAccepted bool      `xml:"EMSignalsAccepted"`
	PowerInfo         PowerInfo `xml:"PowerConsumption>PowerInfo"`
}

// PowerInfo represents SEMP power information
type PowerInfo struct {
	AveragePower      int `xml:"AveragePower"`
	Timestamp         int `xml:"Timestamp"`
	AveragingInterval int `xml:"AveragingInterval"`
}

// DeviceControl represents SEMP device control message
type DeviceControl struct {
	DeviceID                    string `xml:"DeviceId"`
	On                          bool   `xml:"On"`
	RecommendedPowerConsumption int    `xml:"RecommendedPowerConsumption"`
	Timestamp                   int    `xml:"Timestamp"`
}

// PlanningRequest represents SEMP planning request
type PlanningRequest struct {
	Timeframe []Timeframe `xml:"Timeframe"`
}

// Timeframe represents SEMP timeframe
type Timeframe struct {
	DeviceID            string `xml:"DeviceId"`
	EarliestStart       int    `xml:"EarliestStart"`
	LatestEnd           int    `xml:"LatestEnd"`
	MinRunningTime      *int   `xml:"MinRunningTime,omitempty"`
	MaxRunningTime      *int   `xml:"MaxRunningTime,omitempty"`
	MinEnergy           *int   `xml:"MinEnergy,omitempty"`
	MaxEnergy           *int   `xml:"MaxEnergy,omitempty"`
	MaxPowerConsumption *int   `xml:"MaxPowerConsumption,omitempty"`
	MinPowerConsumption *int   `xml:"MinPowerConsumption,omitempty"`
}

// Device2EM represents the device to energy manager message
type Device2EM struct {
	XMLName         xml.Name          `xml:"Device2EM"`
	Xmlns           string            `xml:"xmlns,attr"`
	DeviceInfo      []DeviceInfo      `xml:"DeviceInfo,omitempty"`
	DeviceStatus    []DeviceStatus    `xml:"DeviceStatus,omitempty"`
	PlanningRequest []PlanningRequest `xml:"PlanningRequest,omitempty"`
}

// EM2Device represents the energy manager to device message
type EM2Device struct {
	XMLName       xml.Name        `xml:"EM2Device"`
	Xmlns         string          `xml:"xmlns,attr"`
	DeviceControl []DeviceControl `xml:"DeviceControl,omitempty"`
}
