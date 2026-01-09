package shelly

// RpcRequest represents a Shelly Gen2 RPC request
type RpcRequest struct {
	Id     int
	Src    string
	Method string
	Params RpcParams
}

// RpcParams contains RPC request parameters
type RpcParams struct {
	Owner string
	Role  string
	Value any `json:",omitempty"`
}

// EnumResponse represents an enum value response
type EnumResponse struct {
	Value        string
	Source       string
	LastUpdateTs int64
}

// NumberResponse represents a number value response
type NumberResponse struct {
	Value        float64
	Source       string
	LastUpdateTs int64
}

// PhaseData contains voltage, current and power data for a single phase
type PhaseData struct {
	Voltage float64
	Current float64
	Power   float64
}

// PhaseInfoValue contains aggregated phase information
type PhaseInfoValue struct {
	TotalCurrent   float64
	TotalPower     float64
	TotalActEnergy float64
	PhaseA         PhaseData
	PhaseB         PhaseData
	PhaseC         PhaseData
}

// ObjectResponse represents an object value response with phase info
type ObjectResponse struct {
	Value        PhaseInfoValue
	Source       string
	LastUpdateTs int64
}

// Status represents the charger work state
type Status struct {
	WorkState string
}

// PhaseInfo wraps phase information
type PhaseInfo struct {
	Info PhaseInfoValue
}

// ServiceConfigRequest represents a service configuration request
type ServiceConfigRequest struct {
	Id     int
	Src    string
	Method string
	Params ServiceConfigSet
}

// ServiceConfigSet contains service configuration parameters
type ServiceConfigSet struct {
	Id         int
	AutoCharge bool
}
