package api

//go:generate go run github.com/dmarkham/enumer@v1.5.8 -type TariffType -trimprefix TariffType -transform=lower

type TariffType int

const (
	_ TariffType = iota
	TariffTypePriceStatic
	TariffTypePriceDynamic
	TariffTypeCo2
)
