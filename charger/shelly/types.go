package shelly

// RpcRequest represents a Shelly Gen2 RPC request
type RpcRequest struct {
	Id     int       `json:"id"`
	Src    string    `json:"src"`
	Method string    `json:"method"`
	Params RpcParams `json:"params"`
}

// RpcParams contains RPC request parameters
type RpcParams struct {
	Owner string `json:"owner"`
	Role  string `json:"role"`
	Value any    `json:"value,omitempty"`
}

// RpcResponse represents an rpc response
type RpcResponse[T any] struct {
	Value        T      `json:"value"`
	Source       string `json:"source"`
	LastUpdateTs int64  `json:"last_update_ts"`
}

// PhaseData contains voltage, current and power data for a single phase
type PhaseData struct {
	Voltage float64 `json:"voltage"`
	Current float64 `json:"current"`
	Power   float64 `json:"power"`
}

// PhaseInfo contains aggregated phase information
type PhaseInfo struct {
	TotalCurrent   float64
	TotalPower     float64
	TotalActEnergy float64
	PhaseA         PhaseData
	PhaseB         PhaseData
	PhaseC         PhaseData
}

// Status represents the charger work state
type Status struct {
	WorkState string
}

// ServiceConfigRequest represents a service configuration request
type ServiceConfigRequest struct {
	Id     int              `json:"id"`
	Src    string           `json:"src"`
	Method string           `json:"method"`
	Params ServiceConfigSet `json:"params"`
}

// ServiceConfigSet contains service configuration parameters
type ServiceConfigSet struct {
	Id         int  `json:"id"`
	AutoCharge bool `json:"auto_charge"`
}
