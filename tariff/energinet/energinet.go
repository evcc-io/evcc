package energinet

import (
	//"encoding/json"
	//"time"
)

const URI = "https://api.energidataservice.dk/dataset/Elspotprices?offset=0&start=%s&end=%s&filter={\"PriceArea\":[\"%s\"]}&timezone=dk&limit=48"

type Prices struct {
	Records []PriceInfo `json:"records"`
}

type PriceInfo struct {
	HourUTC string `json:"HourUTC"`
	HourDK string `json:"HourDK"`
	PriceArea string `json:"PriceArea"`
	SpotPriceDKK float64 `json:"SpotPriceDKK"`
	SpotPriceEUR float64 `json:"SpotPriceEUR"`
}

