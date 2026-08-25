package templates

type Usage int

//go:generate go tool enumer -type Usage -trimprefix Usage -transform=lower -text
const (
	UsageGrid Usage = iota
	UsageCurtail
	UsagePV
	UsageBattery
	UsageCharge
	UsageAux
)
