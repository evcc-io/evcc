package nordpool

import (
	"time"
)

const (
	BaseURL    string = "https://dataportal-api.nordpoolgroup.com/api/DayAheadPrices"
	TimeFormat string = "2006-01-02T15:04" // RFC3339 short
)

// TODO: Needed?
type Prices struct {
	Records []PriceInfo `json:"records"`
}

// TODO: Needed?
type PriceInfo struct {
	HourUTC  string
	HourCET  string
	Area     string
	Currency string
	Price    float64
}

type Date struct {
	Date time.Time
}

type AreaEntry struct {
	Start time.Time          `json:"deliveryStart"`
	Stop  time.Time          `json:"deliveryEnd"`
	Entry map[string]float32 `json:"entryPerArea"`
}

type AreaState struct {
	State string   `json:"state"`
	Areas []string `json:"areas"`
}

type AreaAverage struct {
	AreaCode string  `json:"areaCode"`
	Price    float32 `json:"price"`
}

type AveragePrice struct {
	Average float32 `json:"average"`
	Min     float32 `json:"min"`
	Max     float32 `json:"max"`
}

type PriceAggregate struct {
	BlockName     string                  `json:"blockName"`
	Start         time.Time               `json:"deliveryStart"`
	Stop          time.Time               `json:"deliveryEnd"`
	AveragePrices map[string]AveragePrice `json:"averagePricePerArea"`
}

type NordpoolResponse struct {
	RequestDay      Date             `json:"deliveryDateCET"`
	APIVersion      int32            `json:"version"`
	UpdatedAt       time.Time        `json:"updatedAt"`
	DeliveredAreas  []string         `json:"deliveryAreas"`
	Market          string           `json:"market"`
	AreaEntries     []AreaEntry      `json:"multiAreaEntries"`
	PriceAggregates []PriceAggregate `json:"blockPriceAggregates"`
	Currency        string           `json:"currency"`
	ExchangeRate    float32          `json:"exchangeRate"`
	AreaStates      []AreaState      `json:"areaStates"`
	AreaAverages    []AreaAverage    `json:"areaAverages"`
}
