package templates

type ParamType int

//go:generate enumer -type ParamType -trimprefix Type -text
const (
	TypeString ParamType = iota // default type string
	TypeBool
	TypeChoice
	TypeChargeModes
	TypeDuration
	TypeFloat
	TypeNumber
	TypeStringList
)
