package templates

type Class int

//go:generate go run github.com/dmarkham/enumer@v1.5.8 -type Class
const (
	_ Class = iota
	Charger
	Meter
	Vehicle
)
