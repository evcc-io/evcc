package api

type Reason int

//go:generate go tool enumer -type Reason -trimprefix Reason -transform=lower
const (
	ReasonUnknown Reason = iota
	ReasonWaitingForAuthorization
	ReasonDisconnectRequired
)
