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
	// 	msgRxCount += 1

	// 	# When the sendTWCMsg web command is used to send a message to the
	// 	# TWC, it sets lastTWCResponseMsg = b''.  When we see that here,
	// 	# set lastTWCResponseMsg to any unusual message received in response
	// 	# to the sent message.  Never set lastTWCResponseMsg to a commonly
	// 	# repeated message like master or slave linkready, heartbeat, or
	// 	# voltage/kWh report.
	// 	if(lastTWCResponseMsg == b''
	// 		and msg[0:2] != b'\xFB\xE0' and msg[0:2] != b'\xFD\xE0'
	// 		and msg[0:2] != b'\xFC\xE1' and msg[0:2] != b'\xFB\xE2'
	// 		and msg[0:2] != b'\xFD\xE2' and msg[0:2] != b'\xFB\xEB'
	// 		and msg[0:2] != b'\xFD\xEB' and msg[0:2] != b'\xFD\xE0'
	// 	):
	// 		lastTWCResponseMsg = msg

	// 	if(debugLevel >= 9):
	// 		print("Rx@" + time_now() + ": (" + hex_str(ignoredData) + ') ' \
	// 				+ hex_str(msg) + "")

	// 	ignoredData = bytearray()

	// 	# After unescaping special values and removing the leading and
	// 	# trailing C0 bytes, the messages we know about are always 14 bytes
	// 	# long in original TWCs, or 16 bytes in newer TWCs (protocolVersion
	// 	# == 2).
	// 	if(len(msg) != 14 and len(msg) != 16 and len(msg) != 20):
	// 		# In firmware 4.5.3, FD EB (kWh and voltage report), FD ED, FD
	// 		# EE, FD EF, FD F1, and FB A4 messages are length 20 while most
	// 		# other messages are length 16. I'm not sure if there are any
	// 		# length 14 messages remaining.
	// 		print(time_now() + ": ERROR: Ignoring message of unexpected length %d: %s" % \
	// 				(len(msg), hex_str(msg)))
	// 		continue

	// 	checksumExpected = msg[len(msg) - 1]
	// 	checksum = 0
	// 	for i in range(1, len(msg) - 1):
	// 		checksum += msg[i]

	// 	if((checksum & 0xFF) != checksumExpected):
	// 		print("ERROR: Checksum %X does not match %02X.  Ignoring message: %s" %
	// 			(checksum, checksumExpected, hex_str(msg)))
	// 		continue

	// msg length-1 compared to twcmanager as checksum is already removed
	if len(msg) != 13 && len(msg) != 15 && len(msg) != 19 {
		fmt.Println("ignoring message of unexpected length:", len(msg))
		return nil
	}

	// 	if(fakeMaster == 1):
	// 		############################
	// 		# Pretend to be a master TWC

	// 		foundMsgMatch = False

	// 		# We end each regex message search below with \Z instead of $
	// 		# because $ will match a newline at the end of the string or the
	// 		# end of the string (even without the re.MULTILINE option), and
	// 		# sometimes our strings do end with a newline character that is
	// 		# actually the CRC byte with a value of 0A or 0D.

	// 		msgMatch = re.search(b'^\xfd\xe2(..)(.)(..)\x00\x00\x00\x00\x00\x00.+\Z', msg, re.DOTALL)
	// 		if(msgMatch and foundMsgMatch == False):
	// 			# Handle linkready message from slave.
	// 			#
	// 			# We expect to see one of these before we start sending our
	// 			# own heartbeat message to slave.
	// 			# Once we start sending our heartbeat to slave once per
	// 			# second, it should no longer send these linkready messages.
	// 			# If slave doesn't hear master's heartbeat for around 10
	// 			# seconds, it sends linkready once per 10 seconds and starts
	// 			# flashing its red LED 4 times with the top green light on.
	// 			# Red LED stops flashing if we start sending heartbeat
	// 			# again.
	// 			foundMsgMatch = True
	// 			senderID = msgMatch.group(1)
	// 			sign = msgMatch.group(2)
	// 			maxAmps = ((msgMatch.group(3)[0] << 8) + msgMatch.group(3)[1]) / 100

	var slaveMsgType SlaveMessage
	if err := struc.Unpack(bytes.NewBuffer(msg), &slaveMsgType); err != nil {
		panic(err)
	}

	if slaveMsgType.Type == SlaveLinkReadyID {
		var slaveMsg SlaveLinkReady
		if err := struc.Unpack(bytes.NewBuffer(msg), &slaveMsg); err != nil {
			panic(err)
		}
		fmt.Println("SlaveLinkReady:", slaveMsg)

		maxAmps := int(slaveMsg.MaxAmps / 100)
		fmt.Printf("%d amp slave TWC %02X is ready to link", maxAmps, slaveMsg.SenderID)

		if slaveMsg.SenderID == binary.BigEndian.Uint16(fakeTWCID) {
			return fmt.Errorf("slave reports same TWCID as master")
		}

		slaveTWC := h.newSlave(slaveMsg.SenderID, maxAmps)

		// 			if(slaveTWC.protocolVersion == 1 and slaveTWC.minAmpsTWCSupports == 6):
		// 				if(len(msg) == 14):
		// 					slaveTWC.protocolVersion = 1
		// 					slaveTWC.minAmpsTWCSupports = 5
		// 				elif(len(msg) == 16):
		// 					slaveTWC.protocolVersion = 2
		// 					slaveTWC.minAmpsTWCSupports = 6
		// 				if(debugLevel >= 1):
		// 					print(time_now() + ": Set slave TWC %02X%02X protocolVersion to %d, minAmpsTWCSupports to %d." % \
		// 							(senderID[0], senderID[1], slaveTWC.protocolVersion, slaveTWC.minAmpsTWCSupports))

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

		// 			# We expect maxAmps to be 80 on U.S. chargers and 32 on EU
		// 			# chargers. Either way, don't allow
		// 			# slaveTWC.wiringMaxAmps to be greater than maxAmps.
		// 			if(slaveTWC.wiringMaxAmps > maxAmps):
		// 				print("\n\n!!! DANGER DANGER !!!\nYou have set wiringMaxAmpsPerTWC to "
		// 						+ str(wiringMaxAmpsPerTWC)
		// 						+ " which is greater than the max "
		// 						+ str(maxAmps) + " amps your charger says it can handle.  " \
		// 						"Please review instructions in the source code and consult an " \
		// 						"electrician if you don't know what to do.")
		// 				slaveTWC.wiringMaxAmps = maxAmps / 4

		if slaveTWC.wiringMaxAmps > 32 {
			panic("slave wiringMaxAmps too high")
		}

		// 			# Make sure we print one SHB message after a slave
		// 			# linkready message is received by clearing
		// 			# lastHeartbeatDebugOutput. This helps with debugging
		// 			# cases where I can't tell if we responded with a
		// 			# heartbeat or not.
		// 			slaveTWC.lastHeartbeatDebugOutput = ''

		// 			slaveTWC.timeLastRx = time.time()
		// 			slaveTWC.send_master_heartbeat()

		return slaveTWC.sendMasterHeartbeat()
	}

	// 		else:
	// 			msgMatch = re.search(b'\A\xfd\xe0(..)(..)(.......+?).\Z', msg, re.DOTALL)
	// 		if(msgMatch and foundMsgMatch == False):

	// re := regexp.MustCompile(`^\x{fd}\x{e0}(..)(..)(.......+?).$`)
	// if match := re.FindSubmatch(msg); len(match) > 0 {

	if slaveMsgType.Type == SlaveHeartbeatID {
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

		// 			else:
		// 				# I've tried different fakeTWCID values to verify a
		// 				# slave will send our fakeTWCID back to us as
		// 				# receiverID. However, I once saw it send receiverID =
		// 				# 0000.
		// 				# I'm not sure why it sent 0000 and it only happened
		// 				# once so far, so it could have been corruption in the
		// 				# data or an unusual case.
		// 				if(debugLevel >= 1):
		// 					print(time_now() + ": WARNING: Slave TWC %02X%02X status data: " \
		// 							"%s sent to unknown TWC %02X%02X." % \
		// 						(senderID[0], senderID[1],
		// 						hex_str(heartbeatData), receiverID[0], receiverID[1]))

		return fmt.Errorf("slave replied to unexpected master: %02X", receiverID)
	}

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

	// 			if(debugLevel >= 1):
	// 				print(time_now() + ": Slave TWC %02X%02X unexpectedly reported kWh and voltage data: %s." % \
	// 					(senderID[0], senderID[1],
	// 					hex_str(data)))

	// ignore message
	// 	return nil
	// }

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

	fmt.Println("unknown message received")
	return nil
}

func (h *Master) newSlave(slaveID uint16, maxAmps int) *Slave {
	// try:
	//     slaveTWC = slaveTWCs[newSlaveID]
	//     # We didn't get KeyError exception, so this slave is already in
	//     # slaveTWCs and we can simply return it.
	//     return slaveTWC
	// except KeyError:
	//     pass

	// u := binary.BigEndian.Uint16(newSlaveID)
	if slaveTWC, ok := h.slaves[slaveID]; ok {
		return slaveTWC
	}

	// slaveTWC = TWCSlave(newSlaveID, maxAmps)
	// slaveTWCs[newSlaveID] = slaveTWC
	// slaveTWCRoundRobin.append(slaveTWC)

	// if(len(slaveTWCRoundRobin) > 3):
	//     print("WARNING: More than 3 slave TWCs seen on network.  " \
	//         "Dropping oldest: " + hex_str(slaveTWCRoundRobin[0].TWCID) + ".")
	//     delete_slave(slaveTWCRoundRobin[0].TWCID)

	slaveTWC := NewSlave(slaveID, maxAmps)
	h.slaves[slaveID] = slaveTWC

	if len(h.slaves) > 3 {
		panic("too many slaves")
	}

	return slaveTWC
}
