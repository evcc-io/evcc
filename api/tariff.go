package api

//go:generate enumer -type TariffType -trimprefix TariffType -transform=lower -text
//go:generate enumer -type TariffUsage -trimprefix TariffUsage -transform=lower

type TariffType int

const (
	_ TariffType = iota
	TariffTypePriceStatic
	TariffTypePriceDynamic
	TariffTypePriceForecast
	TariffTypeCo2
	TariffTypeSolar
)

type TariffUsage int

const (
	_ TariffUsage = iota
	TariffUsageGrid
	TariffUsageFeedin
	TariffUsageCo2
	TariffUsagePlanner
	TariffUsageSolar
)
