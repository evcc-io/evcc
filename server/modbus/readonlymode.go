package modbus

// go:generate enumer -type ReadOnlyMode -trimprefix ReadOnly -transform=lower

type ReadOnlyMode int

const (
	ReadOnlyFalse ReadOnlyMode = iota
	ReadOnlyDeny
	ReadOnlyTrue
)
