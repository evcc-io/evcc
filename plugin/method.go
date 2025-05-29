package plugin

//go:generate go tool enumer -type Method
type Method int

const (
	_ Method = iota
	Energy
	Power
	Soc
)
