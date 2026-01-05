package amber

import "fmt"

const (
	// ForecastIntervals represents 72 hours of 5-minute intervals for price forecasting
	// This is an "up to" figure, so it should still be OK for 30-minute billing customers
	ForecastIntervals = 864 // 72 hours * 12 intervals per hour
)

var URI = fmt.Sprintf("https://api.amber.com.au/v1/sites/%%s/prices/current?next=%d", ForecastIntervals)

type AdvancedPrice struct {
	Low       float64 `json:"low"`
	Predicted float64 `json:"predicted"`
	High      float64 `json:"high"`
}

type PriceInfo struct {
	Type          string         `json:"type"`
	Date          string         `json:"date"`
	Duration      int            `json:"duration"`
	StartTime     string         `json:"startTime"`
	EndTime       string         `json:"endTime"`
	NemTime       string         `json:"nemTime"`
	PerKwh        float64        `json:"perKwh"`
	Renewables    float64        `json:"renewables"`
	SpotPerKwh    float64        `json:"spotPerKwh"`
	ChannelType   string         `json:"channelType"`
	SpikeStatus   string         `json:"spikeStatus"`
	Descriptor    string         `json:"descriptor"`
	Estimate      bool           `json:"estimate"`
	AdvancedPrice *AdvancedPrice `json:"advancedPrice,omitempty"`
}
