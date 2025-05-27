package plugin

//go:generate go tool enumer -type Key
type Key int

const (
	_ Key = iota
	Power
	Energy
)
