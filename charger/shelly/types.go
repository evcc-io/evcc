package shelly

// RpcRequest represents a Shelly Gen2 RPC request
type RpcRequest struct {
	Id     int    `json:"id"`
	Src    string `json:"src"`
	Method string `json:"method"`
	Params any    `json:"params"`
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
	TotalCurrent   float64   `json:"total_current"`
	TotalPower     float64   `json:"total_power"`
	TotalActEnergy float64   `json:"total_act_energy"`
	PhaseA         PhaseData `json:"phase_a"`
	PhaseB         PhaseData `json:"phase_b"`
	PhaseC         PhaseData `json:"phase_c"`
}
