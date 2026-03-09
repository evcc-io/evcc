package ocpp

type StationStatus int

//go:generate go tool enumer -type StationStatus -trimprefix StationStatus -transform=lower -json
const (
	StationStatusUnknown StationStatus = iota
	StationStatusConfigured
	StationStatusConnected
)
