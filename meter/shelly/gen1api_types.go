package shelly

type Gen1SwitchResponse struct {
	Ison bool
}

type Gen1Status struct {
	Meters []struct {
		Power          float64
		Current        float64
		Voltage        float64
		Total          float64
		Total_Returned float64
	}
	// Shelly EM meter JSON response
	EMeters []struct {
		Power          float64
		Current        float64
		Voltage        float64
		Total          float64
		Total_Returned float64
	}
}
