package templates

// Protocol is a communication protocol used by a template
type Protocol int

//go:generate go tool enumer -type Protocol -trimprefix Protocol -transform=lower -text
const (
	_ Protocol = iota
	ProtocolModbus
	ProtocolHTTP
	ProtocolWebSocket
	ProtocolMQTT
	ProtocolOCPP
	ProtocolEEBus
	ProtocolOther
)

// presetProtocols maps param presets to the protocol they imply
var presetProtocols = map[string]Protocol{
	"ocpp":  ProtocolOCPP,
	"eebus": ProtocolEEBus,
	"mqtt":  ProtocolMQTT,
}
