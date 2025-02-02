package templates

type Class int

//go:generate enumer -type Class -transform=lower
const (
	_ Class = iota
	Charger
	Meter
	Vehicle
	Tariff
	Loadpoint
	Circuit
)
