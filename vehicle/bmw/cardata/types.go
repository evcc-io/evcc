package cardata

import "time"

type VehicleMapping struct {
	Vin         string
	MappedSince time.Time
	MappingType string
}

type Container struct {
	Name        string    `json:"name"`
	Purpose     string    `json:"purpose"`
	ContainerId string    `json:"containerId"`
	Created     time.Time `json:"created"`
}

type CreateContainer struct {
	Name                 string   `json:"name"`
	Purpose              string   `json:"purpose"`
	TechnicalDescriptors []string `json:"technicalDescriptors"`
}

type TelematicDataPoint struct {
	Timestamp time.Time
	Unit      string
	Value     string
}

type TelematicData struct {
	TelematicData map[string]TelematicDataPoint
}

type StreamingMessage struct {
	Vin       string
	EntityId  string
	Topic     string
	TimeStamp time.Time
	Data      map[string]StreamingData
}

type StreamingData struct {
	TimeStamp time.Time
	Value     any
	Unit      string
}
