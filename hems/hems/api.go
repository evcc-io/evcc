package hems

// API describes the HEMS system interface
type API interface {
	Run()
	ConsumptionLimit() float64
	ProductionLimit() float64
}

type Status struct {
	ConsumptionLimit float64 `json:"consumptionLimit"`
	ProductionLimit  float64 `json:"productionLimit"`
}

func GetStatus(api API) *Status {
	if api == nil {
		return nil
	}
	return &Status{
		ConsumptionLimit: api.ConsumptionLimit(),
		ProductionLimit:  api.ProductionLimit(),
	}
}
