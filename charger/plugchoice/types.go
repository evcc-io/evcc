package plugchoice

import "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"

// ChargerData represents a charger from the API
type ChargerData struct {
	UUID             string                    `json:"uuid"`
	ID               int                       `json:"id"`
	Identity         string                    `json:"identity"`
	Reference        string                    `json:"reference"`
	ConnectionStatus string                    `json:"connection_status"`
	Status           core.ChargePointStatus    `json:"status"`
	Error            core.ChargePointErrorCode `json:"error"`
	ErrorInfo        string                    `json:"error_info"`
	CreatedAt        string                    `json:"created_at"`
	UpdatedAt        string                    `json:"updated_at"`
	Model            struct {
		Vendor string `json:"vendor"`
		Name   string `json:"name"`
	} `json:"model"`
	Connectors []Connector `json:"connectors"`
}

// ChargerListResponse is the response from the /chargers endpoint
type ChargerListResponse struct {
	Data  []ChargerData `json:"data"`
	Links Links         `json:"links"`
	Meta  Meta          `json:"meta"`
}

// Links contains pagination links
type Links struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Prev  string `json:"prev"`
	Next  string `json:"next"`
}

// Meta contains pagination metadata
type Meta struct {
	CurrentPage int    `json:"current_page"`
	From        int    `json:"from"`
	LastPage    int    `json:"last_page"`
	Path        string `json:"path"`
	PerPage     int    `json:"per_page"`
	To          int    `json:"to"`
	Total       int    `json:"total"`
}

// StatusResponse is the connector status response
type StatusResponse struct {
	Data ChargerData `json:"data"`
}

// Connector represents a charging connector
type Connector struct {
	ID          int                       `json:"id"`
	ChargerID   int                       `json:"charger_id"`
	ConnectorID int                       `json:"connector_id"`
	Status      core.ChargePointStatus    `json:"status"`
	Error       core.ChargePointErrorCode `json:"error"`
	ErrorInfo   string                    `json:"error_info"`
	MaxAmperage int                       `json:"max_amperage"`
	CreatedAt   string                    `json:"created_at"`
	UpdatedAt   string                    `json:"updated_at"`
}

// PowerResponse is the power usage response
type PowerResponse struct {
	Timestamp string `json:"timestamp"`
	KW        string `json:"kW"`
	L1        string `json:"L1"`
	L2        string `json:"L2"`
	L3        string `json:"L3"`
}
