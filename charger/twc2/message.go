package twc2

type SlaveMessage struct {
	Type     uint16
	SenderID uint16
}

const (
	SlaveLinkReadyID   uint16 = 0xfde2
	SlaveHeartbeatID   uint16 = 0xfde0
	SlaveConsumptionID uint16 = 0xfdeb
	MasterMode1ID      uint16 = 0xfce1
	MasterMode2ID      uint16 = 0xfce2
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

type SlaveConsumption struct {
	Type                 uint16
	SenderID, ReceiverID uint16
	Energy               uint64
	Voltage
}

type Voltage struct {
	L1, L2, L3 uint16
}
