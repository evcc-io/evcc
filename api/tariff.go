package api

//go:generate enumer -type TariffType -trimprefix TariffType -transform=lower -text

type TariffType int

const (
	_ TariffType = iota
	TariffTypePriceStatic
	TariffTypePriceDynamic
	TariffTypePriceForecast
	TariffTypeCo2
)
