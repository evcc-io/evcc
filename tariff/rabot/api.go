package rabot

import "time"

const (
	BaseURI  = "https://api.rabot-charge.de"
	AppToken = "41ee682c-a700-4c6f-85b5-fd912ec4a70d"
)

type LoginResponse struct {
	SessionToken string `json:"sessionToken"`
}

type Contract struct {
	ID string `json:"id"`
}

type ContractsResponse struct {
	Contracts []Contract `json:"contracts"`
}

type PriceValue struct {
	Value float64 `json:"value"`
}

type Record struct {
	Moment     time.Time  `json:"moment"`
	PriceGross PriceValue `json:"priceGross_inCentPerKwh"`
	PriceNet   PriceValue `json:"priceNet_inCentPerKwh"`
}

type PriceResponse struct {
	Records []Record `json:"records"`
}
