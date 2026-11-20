package templates

type ParamType int

//go:generate go tool enumer -type ParamType -trimprefix Type -text
const (
	TypeString ParamType = iota // default type string
	TypeBool
	TypeChoice
	TypeChargeModes
	TypeDuration
	TypeFloat
	TypeInt
	TypeList
	TypeZones
	TypePricePerKWh
	TypeDevice // reference to another configured device by name
)
