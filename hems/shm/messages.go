package shm

import "encoding/xml"

const (
	urnUPNPDevice  = "urn:schemas-upnp-org:device-1-0"
	urnSEMPService = "urn:schemas-simple-energy-management-protocol:service-1-0"
)

// DeviceDescription message definition
type DeviceDescription struct {
	XMLName     xml.Name    `xml:"root"`
	Xmlns       string      `xml:"xmlns,attr"`
	SpecVersion SpecVersion `xml:"specVersion"`
	Device      Device      `xml:"device"`
}

// SpecVersion message definition
type SpecVersion struct {
	Major int `xml:"major"`
	Minor int `xml:"minor"`
}

// Device message definition
type Device struct {
	DeviceType        string            `xml:"deviceType"`
	FriendlyName      string            `xml:"friendlyName"`
	Manufacturer      string            `xml:"manufacturer"`
	ModelName         string            `xml:"modelName"`
	UDN               string            `xml:"UDN"`
	PresentationURL   string            `xml:"presentationURL"`
	ServiceDefinition ServiceDefinition `xml:"semp:X_SEMPSERVICE"`
	ServiceList       []Service         `xml:"serviceList"` // optional
}

// Service message definition
type Service struct {
	ServiceType string `xml:"serviceType"`
	ServiceID   string `xml:"serviceId"`
	SCPDURL     string `xml:"SCPDURL"`
	ControlURL  string `xml:"controlURL"`
	EventSubURL string `xml:"eventSubURL"`
}

// ServiceDefinition message definition
type ServiceDefinition struct {
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

// DeviceInfo message definition
type DeviceInfo struct {
	Identification  Identification
	Characteristics Characteristics
	Capabilities    Capabilities
}

// Identification message definition
type Identification struct {
	DeviceID     string `xml:"DeviceId"`
	DeviceName   string
	DeviceType   string
	DeviceSerial string
	DeviceVendor string
}

// Characteristics message definition
type Characteristics struct {
	MinPowerConsumption int
	MaxPowerConsumption int
	MinOnTime           int `xml:",omitempty"`
	MinOffTime          int `xml:",omitempty"`
}

// method definitions
const (
	MethodMeasurement = "Measurement"
	MethodEstimation  = "Estimation"
)

// Capabilities message definition
type Capabilities struct {
	CurrentPowerMethod   string `xml:"CurrentPower>Method"`
	AbsoluteTimestamps   bool   `xml:"Timestamps>AbsoluteTimestamps"`
	InterruptionsAllowed bool   `xml:"Interruptions>InterruptionsAllowed"`
	OptionalEnergy       bool   `xml:"Requests>OptionalEnergy"`
}

// status definitions
const (
	StatusOn  = "On"
	StatusOff = "Off"
)

// DeviceStatus message definition
type DeviceStatus struct {
	DeviceID          string `xml:"DeviceId"`
	EMSignalsAccepted bool
	Status            string
	PowerInfo         PowerInfo `xml:"PowerConsumption>PowerInfo"`
}

// PowerInfo message definition
type PowerInfo struct {
	AveragePower      int
	Timestamp         int
	AveragingInterval int
}

// PlanningRequest message definition
type PlanningRequest struct {
	Timeframe []Timeframe
}

// Timeframe message definition
type Timeframe struct {
	DeviceID            string `xml:"DeviceId"`
	EarliestStart       int
	LatestEnd           int
	MinRunningTime      *int `xml:",omitempty"`
	MaxRunningTime      *int `xml:",omitempty"`
	MinEnergy           *int `xml:",omitempty"` // AN EVCharger
	MaxEnergy           *int `xml:",omitempty"` // AN EVCharger
	MaxPowerConsumption *int `xml:",omitempty"` // SMA EV CHARGER style
	MinPowerConsumption *int `xml:",omitempty"` // SMA EV CHARGER style
}

// EM2Device is the EM to device message
type EM2Device struct {
	Xmlns         string          `xml:"xmlns,attr"`
	DeviceControl []DeviceControl `xml:",omitempty"`
}

// DeviceControl message definition
type DeviceControl struct {
	DeviceID                    string `xml:"DeviceId"`
	On                          bool
	RecommendedPowerConsumption float64 // AN EVCharger
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
