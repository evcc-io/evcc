package tibber

import "time"

const (
	URI             = "https://api.tibber.com/v1-beta/gql"
	SubscriptionURI = "wss://api.tibber.com/v1-beta/gql/subscriptions"
)

type Home struct {
	ID                string
	TimeZone          string
	Address           Address
	MeteringPointData struct {
		GridCompany string
	}
}

type Address struct {
	Address1, PostalCode, City, Country string
}

type Subscription struct {
	ID        string
	Status    string
	PriceInfo struct {
		Current PriceInfo
		Today   []PriceInfo
		// Tomorrow []PriceInfo
	}
}

type PriceInfo struct {
	Level    string
	StartsAt time.Time
	Total    float64
	// Energy, Tax float64
}

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
