package shelly

// RpcRequest represents a Shelly Gen2 RPC request
type RpcRequest struct {
	Id     int    `json:"id"`
	Src    string `json:"src"`
	Method string `json:"method"`
	Params any    `json:"params"`
}

// RpcResponseWrapper wraps the full RPC response including result
type RpcResponseWrapper[T any] struct {
	Id     int            `json:"id"`
	Src    string         `json:"src"`
	Dst    string         `json:"dst"`
	Result RpcResponse[T] `json:"result"`
}

// RpcResponse represents an rpc response
type RpcResponse[T any] struct {
	Value        T      `json:"value"`
	Source       string `json:"source"`
	LastUpdateTs int64  `json:"last_update_ts"`
}

// PhaseMeasurements contains voltage, current and power data for a single phase
type PhaseMeasurements struct {
	Voltage float64 `json:"voltage"`
	Current float64 `json:"current"`
	Power   float64 `json:"power"`
}

// Measurements contains aggregated phase information
type Measurements struct {
	TotalCurrent   float64           `json:"total_current"`
	TotalPower     float64           `json:"total_power"`
	TotalActEnergy float64           `json:"total_act_energy"`
	PhaseA         PhaseMeasurements `json:"phase_a"`
	PhaseB         PhaseMeasurements `json:"phase_b"`
	PhaseC         PhaseMeasurements `json:"phase_c"`
}

// RoleParams contains parameters for RPC calls with owner and role
type RoleParams struct {
	Owner string `json:"owner"`
	Role  string `json:"role"`
}

// SetValueParams contains parameters for setting a value with owner and role
type SetValueParams struct {
	Owner string  `json:"owner"`
	Role  string  `json:"role"`
	Value float64 `json:"value"`
}
