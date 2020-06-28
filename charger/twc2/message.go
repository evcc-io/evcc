package twc2

type SlaveMessage struct {
	Type     uint16
	SenderID uint16
}

const (
	SlaveLinkReadyID uint16 = 0xfde2
	SlaveHeartbeatID uint16 = 0xfde0
)

type SlaveLinkReady struct {
	Type        uint16
	SenderID    uint16
	Sign        byte
	MaxAmps     uint16
	ZeroPadding [6]byte
}

type SlaveHeartbeat struct {
	Type                 uint16
	SenderID, ReceiverID uint16
	SlaveHeartbeatPayload
}

type SlaveHeartbeatPayload struct {
	State      byte
	AmpsMax    uint16
	AmpsActual uint16
}
