package modbus

// go:generate go tool enumer -type ReadOnlyMode -trimprefix ReadOnly -transform=lower

type ReadOnlyMode int

const (
	ReadOnlyFalse ReadOnlyMode = iota
	ReadOnlyDeny
	ReadOnlyTrue
)
