package v2

import "encoding/json"

// Message is the base WebSocket message format
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// BatteriesData contains battery system status from P1 meter
// Used for battery control responses and power limits
type BatteriesData struct {
	Mode             string  `json:"mode"`               // "zero", "to_full", "standby"
	PowerW           float64 `json:"power_w"`            // Combined battery power
	MaxConsumptionW  float64 `json:"max_consumption_w"`  // Maximum charge power
	MaxProductionW   float64 `json:"max_production_w"`   // Maximum discharge power
}

// AuthRequest is sent by the server requesting authorization
type AuthRequest struct {
	Type string `json:"type"` // "authorization_requested"
	Data struct {
		APIVersion string `json:"api_version"`
	} `json:"data"`
}

// AuthResponse is sent by the client with the authentication token
type AuthResponse struct {
	Type string `json:"type"` // "authorization"
	Data string `json:"data"` // token
}

// AuthConfirm is sent by the server when authorization is successful
type AuthConfirm struct {
	Type string `json:"type"` // "authorized"
}

// Subscribe is sent by the client to subscribe to topics
type Subscribe struct {
	Type string `json:"type"` // "subscribe"
	Data string `json:"data"` // topic name or "*"
}

// ErrorMessage is sent by the server when an error occurs
type ErrorMessage struct {
	Type string `json:"type"` // "error"
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}
