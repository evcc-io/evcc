package ekz

import (
	"time"
)

const URI = "https://api.tariffs.ekz.ch/v1/tariffs"

// TariffResponse represents the JSON response from EKZ API
type TariffResponse struct {
	Prices []PriceEntry `json:"prices"`
}

// PriceEntry represents a single price entry with 15-minute interval
type PriceEntry struct {
	StartTimestamp time.Time `json:"start_timestamp"`
	EndTimestamp   time.Time `json:"end_timestamp"`
	Electricity    []Rate    `json:"electricity"`
	Grid           []Rate    `json:"grid"`
	RegionalFees   []Rate    `json:"regional_fees"`
	Metering       []Rate    `json:"metering"`
	Integrated     []Rate    `json:"integrated"`
}

// Rate represents the pricing information for different components
type Rate struct {
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}
