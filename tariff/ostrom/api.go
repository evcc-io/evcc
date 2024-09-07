package ostrom

import (
	"time"
)

// URIs, production and sandbox
// see https://docs.ostrom-api.io/reference/environments

const (
	URI_AUTH_PRODUCTION = "https://auth.production.ostrom-api.io"
	URI_API_PRODUCTION  = "https://production.ostrom-api.io"
	URI_AUTH_SANDBOX    = "https://auth.sandbox.ostrom-api.io"
	URI_API_SANDBOX     = "https://sandbox.ostrom-api.io"
	URI_AUTH            = URI_AUTH_SANDBOX
	URI_API             = URI_API_SANDBOX
)

const (
	PRODUCT_FAIR     = "SIMPLY_FAIR"
	PRODUCT_FAIR_CAP = "SIMPLY_FAIR_WITH_PRICE_CAP"
	PRODUCT_DYNAMIC  = "SIMPLY_DYNAMIC"
)

type Prices struct {
	Data []ForecastInfo
}

type ForecastInfo struct {
	StartTimestamp time.Time `json:"date"`
	Marketprice    float64   `json:"grossKwhPrice"`
	AdditionalCost float64   `json:"grossKwhTaxAndLevies"`
}

type Contracts struct {
	Data []Contract
}

type Address struct {
	Zip         string `json:"zip"`         //"22083",
	City        string `json:"city"`        //"Hamburg",
	Street      string `json:"street"`      //"Mozartstr.",
	HouseNumber string `json:"housenumber"` //"35"
}

type Contract struct {
	Id        string  `json:"id"`                          //"100523456",
	Type      string  `json:"type"`                        //"ELECTRICITY",
	Product   string  `json:"productCode"`                 //"SIMPLY_DYNAMIC",
	Status    string  `json:"status"`                      //"ACTIVE",
	FirstName string  `json:"customerFirstName"`           //"Max",
	LastName  string  `json:"customerLastName"`            //"Mustermann",
	StartDate string  `json:"startDate"`                   // "2024-03-22",
	Dposit    int     `json:"currentMonthlyDepositAmount"` //120,
	Address   Address `json:"address"`
}
