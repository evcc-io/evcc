package twc2

import (
	"bytes"
	"encoding/binary"

	"github.com/andig/evcc/util"
	"github.com/lunixbochs/struc"
)

var (
	slaveHeartbeatData = []byte{0x01, 0x0F, 0xA0, 0x0F, 0xA0, 0x00, 0x00, 0x00, 0x00}
)

// Slave is a TWC slave instance
type Slave struct {
	log                *util.Logger
	twcID              []byte
	protocolVersion    int
	minAmpsTWCSupports int
	wiringMaxAmps      int
	state              byte
	amps               int
}

// NewSlave creates a new slave instance
func NewSlave(log *util.Logger, slaveID uint16, maxAmps int) *Slave {
	log.DEBUG.Printf("new slave: %4X maxAmps: %d", slaveID, maxAmps)

	h := &Slave{
		log:                log,
		twcID:              make([]byte, 2),
		protocolVersion:    1,
		minAmpsTWCSupports: 6,
		wiringMaxAmps:      maxAmps,
	}
	binary.BigEndian.PutUint16(h.twcID, slaveID)

	return h
}

func (h *Slave) sendMasterHeartbeat() error {
	h.log.TRACE.Println("sendMasterHeartbeat")

	// msg := bytes.NewBuffer([]byte{0xFB, 0xE0})
	// msg.Write(fakeTWCID)
	// msg.Write(h.twcID)
	// msg.Write(h.masterHeartbeatData)

	msg := MasterHeartbeat{
		Header: Header{
			Type:     MasterHeartbeatID,
			SenderID: binary.BigEndian.Uint16(fakeTWCID),
		},
		ReceiverID: binary.BigEndian.Uint16(h.twcID),
		MasterHeartbeatPayload: MasterHeartbeatPayload{
			Command: CmdNOP,
		},
	}

	h.log.TRACE.Printf("slave %4X send heartbeat cmd: %d ampsMax: %d", h.twcID, msg.Command, msg.AmpsMax)

	buf := bytes.NewBuffer(nil)
	if err := struc.Pack(buf, msg); err != nil {
		h.log.ERROR.Println(err)
	}

	return master.send(buf.Bytes())
}

func (h *Slave) receiveHeartbeat(payload SlaveHeartbeatPayload) error {
	h.amps = int(payload.AmpsActual / 100)
	h.state = payload.State

	h.log.TRACE.Printf("slave %4X heartbeat state: %d amps: %d", h.twcID, h.state, h.amps)

	return nil
}
