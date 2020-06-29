package twc2

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/andig/evcc/util"
	"github.com/grid-x/serial"
	"github.com/lunixbochs/struc"
)

const (
	linkDelay      = 100 * time.Millisecond
	loopDelay      = 25 * time.Millisecond
	advertiseDelay = 1 * time.Second
	recvTimeout    = 2 * time.Second
)

var (
	master *Master // singleton instance for sending all data

	fakeTWCID  = []byte{0x77, 0x78}
	masterSign = []byte{0x77}
)

// Master simulates a TWC master instance communicating with the slaves
type Master struct {
	log    *util.Logger
	dev    string
	port   serial.Port
	slaves map[uint16]*Slave
	lastTX time.Time
}

// NewMaster creates TWC master for given serial device
func NewMaster(log *util.Logger, dev string) *Master {
	h := &Master{
		log:    log,
		dev:    dev,
		slaves: make(map[uint16]*Slave),
	}

	// set singleton instance
	if master == nil {
		master = h
	}

	return master
}

// Open opens the serial device with default configuration
func (h *Master) Open() error {
	if h.port == nil {
		port, err := serial.Open(&serial.Config{
			Address:  h.dev,
			BaudRate: 9600,
			DataBits: 8,
			Parity:   "N",
			StopBits: 1,
		})

		if err != nil {
			return err
		}

		h.port = port
	}

	return nil
}

// Close closes the serial port and sets it to nil
func (h *Master) Close() {
	if h.port != nil {
		_ = h.port.Close()
	}
	h.port = nil
}

func (h *Master) send(msg []byte) error {
	msg = Encode(msg)
	h.log.TRACE.Printf("send: % 0X", msg)
	_, err := h.port.Write(msg)
	h.lastTX = time.Now()
	return err
}

func (h *Master) sendLinkReady1() error {
	msg := bytes.NewBuffer([]byte{0xFC, 0xE1})
	msg.Write(fakeTWCID)
	msg.Write(masterSign)
	msg.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	return h.send(msg.Bytes())
}

func (h *Master) sendLinkReady2() error {
	msg := bytes.NewBuffer([]byte{0xFB, 0xE2})
	msg.Write(fakeTWCID)
	msg.Write(masterSign)
	msg.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	return h.send(msg.Bytes())
}

// Run handles the TWC communication
func (h *Master) Run() {
RESTART:
	h.Close()

	numInitMsgsToSend := 10

	for {
		if err := h.Open(); err != nil {
			goto RESTART
		}

		time.Sleep(loopDelay)

		switch {
		// link ready 1
		case numInitMsgsToSend > 5:
			if err := h.sendLinkReady1(); err != nil {
				fmt.Printf("sendLinkReady1: %v\n", err)
				goto RESTART
			}

			numInitMsgsToSend--
			time.Sleep(linkDelay)

		// link ready 2
		case numInitMsgsToSend > 0:
			if err := h.sendLinkReady2(); err != nil {
				fmt.Printf("sendLinkReady2: %v\n", err)
				goto RESTART
			}

			numInitMsgsToSend--
			time.Sleep(linkDelay)

		// master heartbeat
		// TODO send to one slave at a time, use channel?
		case time.Since(h.lastTX) > advertiseDelay:
			for _, slave := range h.slaves {
				if err := slave.sendMasterHeartbeat(); err != nil {
					fmt.Printf("sendMasterHeartbeat: %v\n", err)
					goto RESTART
				}

				time.Sleep(linkDelay)
			}
		}

		if err := h.receive(); err != nil {
			fmt.Printf("receive: %v\n", err)
			goto RESTART
		}
	}
}

func (h *Master) receive() error {
	var msg []byte
	data := make([]byte, 256)

	timeMsgRxStart := time.Now()

	for {
		dataLen, err := h.port.Read(data)
		if err != nil {
			// return error here- this might be a problem with the device
			return err
		}
		h.log.TRACE.Printf("rx   % 0X (%d)", data[0:dataLen], dataLen)

		if dataLen == 0 {
			if len(msg) == 0 {
				return nil
			}

			if time.Since(timeMsgRxStart) > recvTimeout {
				h.log.TRACE.Println("recv: timeout")
				return nil
			}
		}

		// avoid reading in the middle of the message
		if len(msg) == 0 && data[0] != delimiter {
			h.log.TRACE.Println("recv: ignore garbage")
			continue
		}

		timeMsgRxStart = time.Now()
		if len(msg) > 0 && len(msg) < 15 && data[0] == delimiter {
			h.log.TRACE.Println("recv: started in middle of message")
			msg = data[0:dataLen]
			continue
		}

		msg = append(msg, data[0:dataLen]...)
		h.log.TRACE.Printf("recv: % 0X", msg)

		if len(msg) >= 16 && bytes.Count(msg, []byte{delimiter}) > 2 {
			h.log.TRACE.Println("recv: invalid message")
			msg = []byte{}
			continue
		}

		if len(msg) >= 16 {
			// drop final marker after 0xC0
			if msg[len(msg)-2] == delimiter {
				msg = msg[:len(msg)-1]
			}

			msg, err := Decode(msg)
			if err != nil {
				h.log.TRACE.Printf("decode: %v", err)
				return nil
			}

			if err := h.handleMessage(msg); err != nil {
				h.log.TRACE.Printf("handle: %v", err)
			}

			return nil
		}
	}
}

func (h *Master) handleMessage(msg []byte) error {
	// msg length-1 compared to twcmanager as checksum is already removed
	if len(msg) != 13 && len(msg) != 15 && len(msg) != 19 {
		fmt.Println("ignoring message of unexpected length:", len(msg))
		return nil
	}

	var header Header
	if err := struc.Unpack(bytes.NewBuffer(msg), &header); err != nil {
		panic(err)
	}

	switch header.Type {
	case SlaveLinkReadyID:
		var slaveMsg SlaveLinkReady
		if err := struc.Unpack(bytes.NewBuffer(msg), &slaveMsg); err != nil {
			panic(err)
		}

		if slaveMsg.SenderID == binary.BigEndian.Uint16(fakeTWCID) {
			return fmt.Errorf("slave reports same TWCID as master")
		}

		maxAmps := int(slaveMsg.MaxAmps / 100)
		slaveTWC := h.newSlave(slaveMsg.SenderID, maxAmps, len(msg) == 15)

		return slaveTWC.sendMasterHeartbeat()

	case SlaveHeartbeatID:
		var slaveMsg SlaveHeartbeat
		if err := struc.Unpack(bytes.NewBuffer(msg), &slaveMsg); err != nil {
			panic(err)
		}

		slaveTWC, ok := h.slaves[slaveMsg.SenderID]
		if !ok {
			if len(h.slaves) == 0 {
				return fmt.Errorf("slave %04X waiting for registration", slaveMsg.SenderID)
			}
			return fmt.Errorf("slave %04X invalid id", slaveMsg.SenderID)
		}

		if slaveMsg.ReceiverID == binary.BigEndian.Uint16(fakeTWCID) {
			return slaveTWC.receiveHeartbeat(slaveMsg.SlaveHeartbeatPayload)
		}

		return fmt.Errorf("slave %04X replied to unexpected master %04X", slaveMsg.SenderID, slaveMsg.ReceiverID)

	case SlaveConsumptionID:
		var slaveMsg SlaveConsumption
		if err := struc.Unpack(bytes.NewBuffer(msg), &slaveMsg); err != nil {
			panic(err)
		}
		h.log.DEBUG.Printf("slave %04X consumption: %d voltage: %v", slaveMsg.SenderID, slaveMsg.Energy, slaveMsg.Voltage)

		break

	case MasterMode1ID, MasterMode2ID:
		h.log.ERROR.Println("TWC is set to master mode and cannot be controller")

	default:
		h.log.TRACE.Printf("recv: unknown message %4X", header.Type)
	}

	return nil
}

func (h *Master) newSlave(slaveID uint16, maxAmps int, isProtocol2 bool) *Slave {
	if slaveTWC, ok := h.slaves[slaveID]; ok {
		return slaveTWC
	}

	slaveTWC := NewSlave(h.log, slaveID, maxAmps)
	if isProtocol2 {
		slaveTWC.protocolVersion = 2
	}

	h.slaves[slaveID] = slaveTWC

	if cnt := len(h.slaves); cnt > 3 {
		h.log.ERROR.Printf("too many slaves: %d", cnt)
	}

	return slaveTWC
}
