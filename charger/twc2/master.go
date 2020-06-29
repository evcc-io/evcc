package twc2

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

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
	// singleton instance for sending all data
	master *Master

	fakeTWCID  = []byte{0x77, 0x78}
	masterSign = []byte{0x77}
)

type Master struct {
	dev    string
	port   serial.Port
	slaves map[uint16]*Slave
	lastTX time.Time
}

// NewMaster creates TWC master for given serial device
func NewMaster(dev string) *Master {
	h := &Master{
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
		fmt.Println("open", h.dev)

		port, err := serial.Open(&serial.Config{
			Address:  h.dev,
			BaudRate: 9600,
			DataBits: 8,
			Parity:   "N",
			StopBits: 1,
			// RS485:    serial.RS485Config{Enabled: true},
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
		println("close")
		_ = h.port.Close()
	}
	h.port = nil
}

func (h *Master) send(msg []byte) error {
	msg = Encode(msg)
	fmt.Printf("send: % 0X\n", msg)
	_, err := h.port.Write(msg)
	h.lastTX = time.Now()
	return err
}

func (h *Master) sendMasterLinkReady1() error {
	msg := bytes.NewBuffer([]byte{0xFC, 0xE1})
	msg.Write(fakeTWCID)
	msg.Write(masterSign)
	msg.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	return h.send(msg.Bytes())
}

func (h *Master) sendMasterLinkReady2() error {
	msg := bytes.NewBuffer([]byte{0xFB, 0xE2})
	msg.Write(fakeTWCID)
	msg.Write(masterSign)
	msg.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	return h.send(msg.Bytes())
}

func (h *Master) Run() {
RESTART:
	h.Close()

	numInitMsgsToSend := 10

	// m, err := Decode([]byte{0xC0, 0xFD, 0xE0, 0x66, 0x17, 0x77, 0x77, 0x09, 0x06, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x9A, 0xC0})
	// fmt.Printf("% 0x\n", m)
	// panic(err)

	for {
		if err := h.Open(); err != nil {
			fmt.Printf("open: %v\n", err)
			goto RESTART
		}

		time.Sleep(loopDelay)

		if numInitMsgsToSend > 5 {
			// link ready 1
			println("sendMasterLinkReady1")

			if err := h.sendMasterLinkReady1(); err != nil {
				fmt.Printf("sendMasterLinkReady1: %v\n", err)
				goto RESTART
			}

			numInitMsgsToSend--
			time.Sleep(linkDelay)
		} else if numInitMsgsToSend > 0 {
			// link ready 2
			println("sendMasterLinkReady2")

			if err := h.sendMasterLinkReady2(); err != nil {
				fmt.Printf("sendMasterLinkReady2: %v\n", err)
				goto RESTART
			}

			numInitMsgsToSend--
			time.Sleep(linkDelay)
		} else if time.Since(h.lastTX) > advertiseDelay {
			// master heartbeat
			// TODO send to one slave at a time, use channel?
			for _, slave := range h.slaves {
				println("sendMasterHeartbeat")

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

		if dataLen == 0 {
			if len(msg) == 0 {
				return nil
			}

			if time.Since(timeMsgRxStart) > recvTimeout {
				fmt.Println("recv: timeout")
				return nil
			}
		}

		if len(msg) == 0 && data[0] != delimiter {
			continue
		}

		timeMsgRxStart = time.Now()
		if len(msg) > 0 && len(msg) < 15 && data[0] == delimiter {
			fmt.Println("recv: started in middle of message- should not happen")
			msg = data[0:dataLen]
			continue
		}

		msg = append(msg, data[0:dataLen]...)
		fmt.Printf("recv: % 0X\n", msg)

		if len(msg) >= 16 && bytes.Count(msg, []byte{delimiter}) > 2 {
			fmt.Println("recv: invalid message- ignoring")
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
				fmt.Printf("decode: %v\n", err)
				return nil
			}

			if err := h.handleMessage(msg); err != nil {
				fmt.Printf("handle: %v\n", err)
			}

			return nil
		}
	}
}

func (h *Master) handleMessage(msg []byte) error {
	fmt.Printf("handle: % 0x\n", msg)

	// msg length-1 compared to twcmanager as checksum is already removed
	if len(msg) != 13 && len(msg) != 15 && len(msg) != 19 {
		fmt.Println("ignoring message of unexpected length:", len(msg))
		return nil
	}

	var slaveMsgType SlaveMessage
	if err := struc.Unpack(bytes.NewBuffer(msg), &slaveMsgType); err != nil {
		panic(err)
	}

	switch slaveMsgType.Type {
	case SlaveLinkReadyID:
		var slaveMsg SlaveLinkReady
		if err := struc.Unpack(bytes.NewBuffer(msg), &slaveMsg); err != nil {
			panic(err)
		}
		fmt.Println("SlaveLinkReady:", slaveMsg)

		if slaveMsg.SenderID == binary.BigEndian.Uint16(fakeTWCID) {
			return fmt.Errorf("slave reports same TWCID as master")
		}

		maxAmps := int(slaveMsg.MaxAmps / 100)
		slaveTWC := h.newSlave(slaveMsg.SenderID, maxAmps)

		// msg length-1 compared to twcmanager as checksum is already removed
		if slaveTWC.protocolVersion == 1 && slaveTWC.minAmpsTWCSupports == 6 {
			if len(msg) == 13 {
				slaveTWC.protocolVersion = 1
				slaveTWC.minAmpsTWCSupports = 5
			} else if len(msg) == 15 {
				slaveTWC.protocolVersion = 2
				slaveTWC.minAmpsTWCSupports = 6
			}
		}

		return slaveTWC.sendMasterHeartbeat()

	case SlaveHeartbeatID:
		var slaveMsg SlaveHeartbeat
		if err := struc.Unpack(bytes.NewBuffer(msg), &slaveMsg); err != nil {
			panic(err)
		}
		fmt.Println("SlaveHeartbeat:", slaveMsg)

		slaveTWC, ok := h.slaves[slaveMsg.SenderID]
		if !ok {
			return fmt.Errorf("invalid slave id: %02X", slaveMsg.SenderID)
		}

		if slaveMsg.ReceiverID == binary.BigEndian.Uint16(fakeTWCID) {
			return slaveTWC.receiveSlaveHeartbeat(slaveMsg.SlaveHeartbeatPayload)
		}

		return fmt.Errorf("slave replied to unexpected master: %02X", slaveMsg.ReceiverID)

	// re = regexp.MustCompile(`^\x{fd}\x{eb}(..)(..)(.+?).$`)
	// if match := re.FindSubmatch(msg); len(match) > 0 {

	// 			# Handle kWh total and voltage message from slave.
	// 			#
	// 			# This message can only be generated by TWCs running newer
	// 			# firmware.  I believe it's only sent as a response to a
	// 			# message from Master in this format:
	// 			#   FB EB <Master TWCID> <Slave TWCID> 00 00 00 00 00 00 00 00 00
	// 			# Since we never send such a message, I don't expect a slave
	// 			# to ever send this message to us, but we handle it just in
	// 			# case.
	// 			# According to FuzzyLogic, this message has the following
	// 			# format on an EU (3-phase) TWC:
	// 			#   FD EB <Slave TWCID> 00000038 00E6 00F1 00E8 00
	// 			#   00000038 (56) is the total kWh delivered to cars
	// 			#     by this TWC since its construction.
	// 			#   00E6 (230) is voltage on phase A
	// 			#   00F1 (241) is voltage on phase B
	// 			#   00E8 (232) is voltage on phase C
	// 			#
	// 			# I'm guessing in world regions with two-phase power that
	// 			# this message would be four bytes shorter, but the pattern
	// 			# above will match a message of any length that starts with
	// 			# FD EB.
	// 			foundMsgMatch = True
	// 			senderID = msgMatch.group(1)
	// 			receiverID = msgMatch.group(2)
	// 			data = msgMatch.group(3)

	case SlaveConsumptionID:
		var slaveMsg SlaveConsumption
		if err := struc.Unpack(bytes.NewBuffer(msg), &slaveMsg); err != nil {
			panic(err)
		}
		fmt.Println("SlaveConsumption:", slaveMsg)

		break

	case MasterMode1ID, MasterMode2ID:
		fmt.Println("TWC is set to master mode and cannot be controller")

	default:
		fmt.Println("unknown message received")
	}

	return nil
}

func (h *Master) newSlave(slaveID uint16, maxAmps int) *Slave {
	if slaveTWC, ok := h.slaves[slaveID]; ok {
		return slaveTWC
	}

	slaveTWC := NewSlave(slaveID, maxAmps)
	h.slaves[slaveID] = slaveTWC

	if len(h.slaves) > 3 {
		panic("twc2: too many slaves")
	}

	return slaveTWC
}
