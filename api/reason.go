package api

type Reason int

//go:generate enumer -type Reason -trimprefix Reason -transform=lower
const (
	ReasonUnknown Reason = iota
	ReasonWaitingForAuthorization
)
