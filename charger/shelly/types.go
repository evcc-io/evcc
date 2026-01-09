package shelly

// RpcRequest represents a Shelly Gen2 RPC request
type RpcRequest struct {
	Id     int
	Src    string
	Method string
	Params any
}

// RpcParams contains RPC request parameters
type RpcParams struct {
	Owner string
	Role  string
	Value any `json:",omitempty"`
}

// RpcResponse represents an rpc response
type RpcResponse[T any] struct {
	Value        T
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

// PhaseInfo wraps phase information
type PhaseInfo struct {
	Info PhaseInfoValue
}

// ServiceConfigSet contains service configuration parameters
type ServiceConfigSet struct {
	Id         int
	AutoCharge bool
}
