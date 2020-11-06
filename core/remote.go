package core

// RemoteDemand defines external status demand
type RemoteDemand string

// remote status demand definition
const (
	RemoteEnable      RemoteDemand = "enable"
	RemoteHardDisable RemoteDemand = "disable"
	RemoteSoftDisable RemoteDemand = "softdisable"
)
