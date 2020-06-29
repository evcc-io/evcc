package twc2

const (
	// message IDs
	SlaveLinkReadyID   uint16 = 0xfde2
	SlaveHeartbeatID   uint16 = 0xfde0
	SlaveConsumptionID uint16 = 0xfdeb
	MasterMode1ID      uint16 = 0xfce1
	MasterMode2ID      uint16 = 0xfce2
	MasterHeartbeatID  uint16 = 0xfbe0

	// master commands
	CmdNOP         byte = 0x00
	CmdMaxCurrent  byte = 0x05 // protocol 1 command (before charging starts)
	CmdMaxCurrent2 byte = 0x09 // protocol 2 command (before and after charging starts)
)

// Header is the generic message header containing type and sender
type Header struct {
	Type     uint16
	SenderID uint16
}

// SlaveLinkReady is the slave's response to the master's link ready message
type SlaveLinkReady struct {
	Header
	Sign        byte
	MaxAmps     uint16
	ZeroPadding [6]byte
}

// SlaveHeartbeat is the slave's regular heartbeat message
type SlaveHeartbeat struct {
	Header
	ReceiverID uint16
	SlaveHeartbeatPayload
}

// SlaveHeartbeatPayload is the payload for the SlaveHeartbeat message
type SlaveHeartbeatPayload struct {
	State      byte
	AmpsMax    uint16
	AmpsActual uint16
}

// SlaveConsumption is the slave's consumption message
type SlaveConsumption struct {
	Header
	Energy  uint32
	Voltage struct {
		L1, L2, L3 uint16
	}
}

// MasterHeartbeat is the slave's consumption message
type MasterHeartbeat struct {
	Header
	ReceiverID uint16
	MasterHeartbeatPayload
}

// MasterHeartbeatPayload is the payload for the MasterHeartbeat message
type MasterHeartbeatPayload struct {
	Command     byte
	AmpsMax     uint16
	PlugState   byte
	ZeroPadding [3]byte
}
