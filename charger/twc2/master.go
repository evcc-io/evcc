package twc2

import (
	"bytes"
	"encoding/binary"
	"errors"
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

	// timeMsgRxStart = time.time()
	timeMsgRxStart := time.Now()

	// while True:
	// 	now = time.time()
	// 	dataLen = ser.inWaiting()
	for {
		dataLen, err := h.port.Read(data)
		if err != nil {
			fmt.Printf("recv: %v\n", err)
			return err
		}

		fmt.Println("recv:", dataLen)

		// 	if(dataLen == 0):
		// 		if(msgLen == 0):
		// 			# No message data waiting and we haven't received the
		// 			# start of a new message yet. Break out of inner while
		// 			# to continue at top of outer while loop where we may
		// 			# decide to send a periodic message.
		// 			break
		// 		else:
		// 			# No message data waiting but we've received a partial
		// 			# message that we should wait to finish receiving.
		// 			if(now - timeMsgRxStart >= 2.0):
		// 				if(debugLevel >= 9):
		// 					print(time_now() + ": Msg timeout (" + hex_str(ignoredData) +
		// 							') ' + hex_str(msg[0:msgLen]))
		// 				msgLen = 0
		// 				ignoredData = bytearray()
		// 				break
		// 			time.sleep(0.025)
		// 			continue
		// 	else:
		// 		dataLen = 1
		// 		data = ser.read(dataLen)

		if dataLen == 0 {
			if len(msg) == 0 {
				return nil
			}

			if time.Since(timeMsgRxStart) > recvTimeout {
				return errors.New("recv timeout")
			}
		}

		// 	timeMsgRxStart = now
		// 	timeLastRx = now

		timeMsgRxStart = time.Now()
		// timeLastRx := time.Now()

		// 	if(msgLen == 0 and data[0] != 0xc0):
		// 		# We expect to find these non-c0 bytes between messages, so
		// 		# we don't print any warning at standard debug levels.
		// 		if(debugLevel >= 11):
		// 			print("Ignoring byte %02X between messages." % (data[0]))
		// 		ignoredData += data
		// 		continue
		if len(msg) == 0 && data[0] != delimiter {
			continue
		}

		// 	elif(msgLen > 0 and msgLen < 15 and data[0] == 0xc0):
		// 		# If you see this when the program is first started, it
		// 		# means we started listening in the middle of the TWC
		// 		# sending a message so we didn't see the whole message and
		// 		# must discard it. That's unavoidable.
		// 		# If you see this any other time, it means there was some
		// 		# corruption in what we received. It's normal for that to
		// 		# happen every once in awhile but there may be a problem
		// 		# such as incorrect termination or bias resistors on the
		// 		# rs485 wiring if you see it frequently.
		// 		if(debugLevel >= 10):
		// 			print("Found end of message before full-length message received.  " \
		// 					"Discard and wait for new message.")

		// 		msg = data
		// 		msgLen = 1
		// 		continue
		if len(msg) > 0 && len(msg) < 15 && data[0] == delimiter {
			fmt.Println("started in middle of message- should not happen")
			msg = data[0 : dataLen-1]
			continue
		}

		// 	if(msgLen == 0):
		// 		msg = bytearray()
		// 	msg += data
		// 	msgLen += 1
		msg = append(msg, data[0:dataLen-1]...)

		fmt.Printf("recv: % 0X\n", msg)
		// 	# Messages are usually 17 bytes or longer and end with \xc0\xfe.
		// 	# However, when the network lacks termination and bias
		// 	# resistors, the last byte (\xfe) may be corrupted or even
		// 	# missing, and you may receive additional garbage bytes between
		// 	# messages.
		// 	#
		// 	# TWCs seem to account for corruption at the end and between
		// 	# messages by simply ignoring anything after the final \xc0 in a
		// 	# message, so we use the same tactic. If c0 happens to be within
		// 	# the corrupt noise between messages, we ignore it by starting a
		// 	# new message whenever we see a c0 before 15 or more bytes are
		// 	# received.
		// 	#
		// 	# Uncorrupted messages can be over 17 bytes long when special
		// 	# values are "escaped" as two bytes. See notes in send_msg.
		// 	#
		// 	# To prevent most noise between messages, add a 120ohm
		// 	# "termination" resistor in parallel to the D+ and D- lines.
		// 	# Also add a 680ohm "bias" resistor between the D+ line and +5V
		// 	# and a second 680ohm "bias" resistor between the D- line and
		// 	# ground. See here for more information:
		// 	#   https://www.ni.com/support/serial/resinfo.htm
		// 	#   http://www.ti.com/lit/an/slyt514/slyt514.pdf
		// 	# This explains what happens without "termination" resistors:
		// 	#   https://e2e.ti.com/blogs_/b/analogwire/archive/2016/07/28/rs-485-basics-when-termination-is-necessary-and-how-to-do-it-properly
		// 	if(msgLen >= 16 and data[0] == 0xc0):
		// 		break
		// if len(msg) >= 16 && data[0] == delimiter {
		// 	println("corrupted - ignoring")
		// 	return nil
		// }

		// if(msgLen >= 16):
		// 	msg = unescape_msg(msg, msgLen)
		// 	# Set msgLen = 0 at start so we don't have to do it on errors below.
		// 	# len($msg) now contains the unescaped message length.
		// 	msgLen = 0
		if len(msg) >= 16 {
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

	// 		else:
	// 			msgMatch = re.search(b'\A\xfd\xeb(..)(..)(.+?).\Z', msg, re.DOTALL)
	// 		if(msgMatch and foundMsgMatch == False):

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

	// 		else:
	// 			msgMatch = re.search(b'\A\xfc(\xe1|\xe2)(..)(.)\x00\x00\x00\x00\x00\x00\x00\x00.+\Z', msg, re.DOTALL)
	// 		if(msgMatch and foundMsgMatch == False):

	// re = regexp.MustCompile(`^\x{fc}(\x{e1}|\x{e2})(..)(.)\x{00}\x{00}\x{00}\x{00}\x{00}\x{00}\x{00}\x{00}.+$`)
	// if match := re.FindSubmatch(msg); len(match) > 0 {

	// 			foundMsgMatch = True
	// 			print(time_now() + " ERROR: TWC is set to Master mode so it can't be controlled by TWCManager.  " \
	// 					"Search installation instruction PDF for 'rotary switch' and set " \
	// 					"switch so its arrow points to F on the dial.")

	// 	panic("TWC is set to master mode and cannot be controlled")
	// }

	// 		if(foundMsgMatch == False):
	// 			print(time_now() + ": *** UNKNOWN MESSAGE FROM SLAVE:" + hex_str(msg)
	// 					+ "\nPlease private message user CDragon at http://teslamotorsclub.com " \
	// 					"with a copy of this error.")
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
