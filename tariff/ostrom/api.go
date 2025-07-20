package ostrom

import (
	"time"
)

// URIs, production and sandbox
// see https://docs.ostrom-api.io/reference/environments

const (
	URI_AUTH_PRODUCTION  = "https://auth.production.ostrom-api.io"
	URI_API_PRODUCTION   = "https://production.ostrom-api.io"
	URI_AUTH_SANDBOX     = "https://auth.sandbox.ostrom-api.io"
	URI_API_SANDBOX      = "https://sandbox.ostrom-api.io"
	URI_GET_CITYID       = "https://api.ostrom.de/v1/addresses/cities"
	URI_GET_STATIC_PRICE = "https://api.ostrom.de/v1/tariffs/city-id"
	URI_AUTH             = URI_AUTH_PRODUCTION
	URI_API              = URI_API_PRODUCTION
)

const (
	PRODUCT_FAIR     = "SIMPLY_FAIR"
	PRODUCT_FAIR_CAP = "SIMPLY_FAIR_WITH_PRICE_CAP"
	PRODUCT_DYNAMIC  = "SIMPLY_DYNAMIC"
	PRODUCT_BASIC    = "basisProdukt"
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
	Id        int64   `json:"id"`                          //"100523456",
	Type      string  `json:"type"`                        //"ELECTRICITY",
	Product   string  `json:"productCode"`                 //"SIMPLY_DYNAMIC",
	Status    string  `json:"status"`                      //"ACTIVE",
	FirstName string  `json:"customerFirstName"`           //"Max",
	LastName  string  `json:"customerLastName"`            //"Mustermann",
	StartDate string  `json:"startDate"`                   // "2024-03-22",
	Dposit    int     `json:"currentMonthlyDepositAmount"` //120,
	Address   Address `json:"address"`
}

type CityId []struct {
	Id       int    `json:"id"`
	Postcode string `json:"postcode"`
	Name     string `json:"name"`
}

type Tariffs struct {
	Ostrom []struct {
		ProductCode                              string  `json:"productCode"`
		Tariff                                   int     `json:"tariff"`
		BasicFee                                 int     `json:"basicFee"`
		NetworkFee                               float64 `json:"networkFee"`
		UnitPricePerkWH                          float64 `json:"unitPricePerkWH"`
		TariffWithStormPreisBremse               int     `json:"tariffWithStormPreisBremse"`
		StromPreisBremseUnitPrice                int     `json:"stromPreisBremseUnitPrice"`
		AccumulatedUnitPriceWithStromPreisBremse float64 `json:"accumulatedUnitPriceWithStromPreisBremse"`
		UnitPrice                                float64 `json:"unitPrice"`
		EnergyConsumption                        int     `json:"energyConsumption"`
		BasePriceBrutto                          float64 `json:"basePriceBrutto"`
		WorkingPriceBrutto                       float64 `json:"workingPriceBrutto"`
		WorkingPriceNetto                        float64 `json:"workingPriceNetto"`
		MeterChargeBrutto                        int     `json:"meterChargeBrutto"`
		WorkingPricePowerTax                     float64 `json:"workingPricePowerTax"`
		AverageHourlyPriceToday                  float64 `json:"averageHourlyPriceToday,omitempty"`
		MinHourlyPriceToday                      float64 `json:"minHourlyPriceToday,omitempty"`
		MaxHourlyPriceToday                      float64 `json:"maxHourlyPriceToday,omitempty"`
	} `json:"ostrom"`
	Footprint struct {
		Usage          int `json:"usage"`
		KgCO2Emissions int `json:"kgCO2Emissions"`
	} `json:"footprint"`
	IsPendingApplicationAllowed bool   `json:"isPendingApplicationAllowed"`
	Status                      string `json:"status"`
	PartnerName                 any    `json:"partnerName"`
}
