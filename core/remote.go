package core

// RemoteDemand defines external status demand
type RemoteDemand string

// remote status demand definition
const (
	RemoteEnable      RemoteDemand = ""
	RemoteHardDisable RemoteDemand = "hard"
	RemoteSoftDisable RemoteDemand = "soft"
)
