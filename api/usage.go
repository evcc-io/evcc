package api

type Usage int

//go:generate enumer -type Usage -trimprefix Usage -transform=lower -text
const (
	UsageGrid Usage = iota + 1
	UsagePV
	UsageBattery
	UsageCharge
	UsageAux
	UsageVehicle // maps parameter to class for defaults.yaml
)
