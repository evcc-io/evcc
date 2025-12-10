package hems

// API describes the HEMS system interface
type API interface {
	Run()
	MaxPower() float64
}

type Status struct {
	MaxPower float64
}

func GetStatus(api API) *Status {
	if api == nil {
		return nil
	}
	return &Status{
		MaxPower: api.MaxPower(),
	}
}
