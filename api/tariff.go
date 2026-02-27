package api

//go:generate go tool enumer -type TariffType -trimprefix TariffType -transform=lower -text
//go:generate go tool enumer -type TariffUsage -trimprefix TariffUsage -transform=lower

type TariffType int

const (
	_ TariffType = iota
	TariffTypePriceStatic
	TariffTypePriceDynamic
	TariffTypePriceForecast
	TariffTypeCo2
	TariffTypeSolar
	TariffTypeWeather // outdoor temperature forecast in °C
)

type TariffUsage int

const (
	_ TariffUsage = iota
	TariffUsageCo2
	TariffUsageFeedIn
	TariffUsageGrid
	TariffUsagePlanner
	TariffUsageSolar
	TariffUsageWeather // outdoor temperature forecast
)
