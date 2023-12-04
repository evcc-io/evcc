package energinet

const (
	URI        = "https://api.energidataservice.dk/dataset/Elspotprices?offset=0&start=%s&end=%s&filter={\"PriceArea\":[\"%s\"]}&timezone=dk&limit=48"
	TimeFormat = "2006-01-02T15:04" // RFC3339 short
)

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
