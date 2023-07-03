package energinet

const URI = "https://api.energidataservice.dk/dataset/Elspotprices?offset=0&start=%s&end=%s&filter={\"PriceArea\":[\"%s\"]}&timezone=dk&limit=48"

type Prices struct {
	Records []PriceInfo `json:"records"`
}

type PriceInfo struct {
	HourUTC      string
	HourDK       string
	PriceArea    string
	SpotPriceDKK float64
	SpotPriceEUR float64
}
