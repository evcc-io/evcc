package hems

// API describes the HEMS system interface
type API interface {
	Run()
	ConsumptionLimit() float64
}

type Status struct {
	ConsumptionLimit float64 `json:"consumptionLimit"`
	// FeedinLimit      float64 `json:"feedinLimit,omitempty"`
}

func GetStatus(api API) *Status {
	if api == nil {
		return nil
	}
	return &Status{
		ConsumptionLimit: api.ConsumptionLimit(),
	}
}
