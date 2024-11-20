package autonomic

import "time"

type IntValue struct {
	UpdateTime time.Time
	Value      int
}
type FloatValue struct {
	UpdateTime time.Time
	Value      float64
}
type StringValue struct {
	UpdateTime time.Time
	Value      string
}

type MetricsResponse struct {
	Metrics struct {
		Position struct {
			Value struct {
				Location struct {
					Lat, Lon float64
				}
			}
		}
		Odometer                FloatValue
		XevPlugChargerStatus    StringValue
		XevBatteryRange         FloatValue
		XevBatteryStateOfCharge FloatValue
	}
}
