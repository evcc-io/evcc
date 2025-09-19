package bmw

import (
	"time"

	"golang.org/x/oauth2"
)

type GcidIdToken struct {
	*oauth2.Token
	IdToken string `json:"id_token"`
	Gcid    string `json:"gcid"`
}

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
