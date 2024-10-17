package smartenergy

import "time"

const URI = "https://apis.smartenergy.at/market/v1/price"

type Prices struct {
	Data []Price
}

type Price struct {
	Date  time.Time
	Value float64
}
