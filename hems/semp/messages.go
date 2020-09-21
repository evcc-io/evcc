package semp

import "encoding/xml"

const (
	urnUPNPDevice  = "urn:schemas-upnp-org:device-1-0"
	urnSEMPService = "urn:schemas-simple-energy-management-protocol:service-1-0"
)

type DeviceDescription struct {
	XMLName     xml.Name    `xml:"root"`
	Xmlns       string      `xml:"xmlns,attr"`
	SpecVersion SpecVersion `xml:"specVersion"`
	Device      Device      `xml:"device"`
}

type SpecVersion struct {
	Major int `xml:"major"`
	Minor int `xml:"minor"`
}

type Device struct {
	DeviceType      string      `xml:"deviceType"`
	FriendlyName    string      `xml:"friendlyName"`
	Manufacturer    string      `xml:"manufacturer"`
	ModelName       string      `xml:"modelName"`
	UDN             string      `xml:"UDN"`
	PresentationURL string      `xml:"presentationURL"`
	SEMPService     SEMPService `xml:"semp:X_SEMPSERVICE"`
	ServiceList     []Service   `xml:"serviceList"` // optional
}

type Service struct {
	ServiceType string `xml:"serviceType"`
	ServiceID   string `xml:"serviceId"`
	SCPDURL     string `xml:"SCPDURL"`
	ControlURL  string `xml:"controlURL"`
	EventSubURL string `xml:"eventSubURL"`
}

type SEMPService struct {
	Xmlns          string `xml:"xmlns:semp,attr"`
	Server         string `xml:"semp:server"`
	BasePath       string `xml:"semp:basePath"`
	Transport      string `xml:"semp:transport"`
	ExchangeFormat string `xml:"semp:exchangeFormat"`
	WsVersion      string `xml:"semp:wsVersion"`
}

// Device2EM is the device to EM message
type Device2EM struct {
	Xmlns           string            `xml:"xmlns,attr"`
	DeviceInfo      []DeviceInfo      `xml:",omitempty"`
	DeviceStatus    []DeviceStatus    `xml:",omitempty"`
	PlanningRequest []PlanningRequest `xml:",omitempty"`
}

type DeviceInfo struct {
	Identification  Identification
	Characteristics Characteristics
	Capabilities    Capabilities
}

type Identification struct {
	DeviceID     string `xml:"DeviceId"`
	DeviceName   string
	DeviceType   string
	DeviceSerial string
	DeviceVendor string
}

type Characteristics struct {
	MinPowerConsumption int
	MaxPowerConsumption int
	MinOnTime           int `xml:",omitempty"`
	MinOffTime          int `xml:",omitempty"`
}

type Capabilities struct {
	CurrentPower  CurrentPower
	Timestamps    Timestamps
	Interruptions Interruptions
	Requests      Requests
}

const (
	MethodMeasurement = "Measurement"
	MethodEstimation  = "Estimation"
)

type CurrentPower struct {
	Method string
}

type Timestamps struct {
	AbsoluteTimestamps bool
}

type Interruptions struct {
	InterruptionsAllowed bool
}

type Requests struct {
	OptionalEnergy bool
}

const (
	StatusOn  = "On"
	StatusOff = "Off"
)

type DeviceStatus struct {
	DeviceID          string `xml:"DeviceId"`
	EMSignalsAccepted bool
	Status            string
	PowerConsumption  PowerConsumption
}

type PowerConsumption struct {
	PowerInfo PowerInfo
}

type PowerInfo struct {
	AveragePower      int
	Timestamp         int
	AveragingInterval int
}

type PlanningRequest struct {
	Timeframe Timeframe
}

type Timeframe struct {
	DeviceID       string `xml:"DeviceId"`
	EarliestStart  int
	LatestEnd      int
	MinRunningTime *int `xml:",omitempty"`
	MaxRunningTime *int `xml:",omitempty"`
	MinEnergy      *int `xml:",omitempty"` // AN EVCharger
	MaxEnergy      *int `xml:",omitempty"` // AN EVCharger
}

// EM2Device is the EM to device message
type EM2Device struct {
	Xmlns           string            `xml:"xmlns,attr"`
	DeviceControl   []DeviceControl   `xml:",omitempty"`
	PlanningRequest []PlanningRequest `xml:",omitempty"`
}

type DeviceControl struct {
	DeviceID                    string `xml:"DeviceId"`
	On                          bool
	RecommendedPowerConsumption int // AN EVCharger
	Timestamp                   int
}

// Device2EMMsg is the XML message container
func Device2EMMsg() Device2EM {
	msg := Device2EM{
		Xmlns:        "http://www.sma.de/communication/schema/SEMP/v1",
		DeviceInfo:   make([]DeviceInfo, 0),
		DeviceStatus: make([]DeviceStatus, 0),
	}

	return msg
}
