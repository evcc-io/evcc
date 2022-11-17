package tibber

const SubscriptionURI = "wss://api.tibber.com/v1-beta/gql/subscriptions"

type LiveMeasurement struct {
	// Timestamp                       time.Time
	Power                           float64
	LastMeterConsumption            float64
	LastMeterProduction             float64
	CurrentL1, CurrentL2, CurrentL3 float64
	// Currency                        string
	// AccumulatedConsumption          float64
	// AccumulatedCost                 float64
	// MinPower                        float64
	// AveragePower                    float64
	// MaxPower                        float64
}
