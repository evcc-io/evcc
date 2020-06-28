package twc2

import (
	"bytes"
	"fmt"
)

var (
	slaveHeartbeatData = []byte{0x01, 0x0F, 0xA0, 0x0F, 0xA0, 0x00, 0x00, 0x00, 0x00}
)

type Slave struct {
	twcID               []byte
	protocolVersion     int
	minAmpsTWCSupports  int
	wiringMaxAmps       int
	masterHeartbeatData []byte
}

func NewSlave(newSlaveID []byte, maxAmps int) *Slave {
	s := &Slave{
		twcID:               newSlaveID,
		protocolVersion:     1,
		minAmpsTWCSupports:  6,
		wiringMaxAmps:       maxAmps,
		masterHeartbeatData: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	return s
}

// func (h *Slave) sendSlaveHeartbeat(masterID []byte) error {
// 	// if(self.protocolVersion == 1 and len(slaveHeartbeatData) > 7):
// 	//     # Cut array down to length 7
// 	//     slaveHeartbeatData = slaveHeartbeatData[0:7]
// 	// elif(self.protocolVersion == 2):
// 	//     while(len(slaveHeartbeatData) < 9):
// 	//         # Increase array length to 9
// 	//         slaveHeartbeatData.append(0x00)

// 	// send_msg(bytearray(b'\xFD\xE0') + fakeTWCID + bytearray(masterID) + bytearray(slaveHeartbeatData))

// 	if h.protocolVersion == 1 && len(slaveHeartbeatData) > 7 {
// 		slaveHeartbeatData = slaveHeartbeatData[0:6]
// 	}
// 	if h.protocolVersion == 2 {
// 		for len(slaveHeartbeatData) < 9 {
// 			slaveHeartbeatData = append(slaveHeartbeatData, 0x00)
// 		}
// 	}

// 	msg := bytes.NewBuffer([]byte{0xFD, 0xE0})
// 	msg.Write(fakeTWCID)
// 	msg.Write(masterID)
// 	msg.Write(slaveHeartbeatData)

// 	return master.send(msg.Bytes())
// }

func (h *Slave) sendMasterHeartbeat() error {
	fmt.Println("sendMasterHeartbeat")
	// if(len(overrideMasterHeartbeatData) >= 7):
	// 	self.masterHeartbeatData = overrideMasterHeartbeatData

	// if(self.protocolVersion == 2):
	// 	# TODO: Start and stop charging using protocol 2 commands to TWC
	// 	# instead of car api if I ever figure out how.
	// 	if(self.lastAmpsOffered == 0 and self.reportedAmpsActual > 4.0):
	// 		# Car is trying to charge, so stop it via car API.
	// 		# car_api_charge() will prevent telling the car to start or stop
	// 		# more than once per minute. Once the car gets the message to
	// 		# stop, reportedAmpsActualSignificantChangeMonitor should drop
	// 		# to near zero within a few seconds.
	// 		# WARNING: If you own two vehicles and one is charging at home but
	// 		# the other is charging away from home, this command will stop
	// 		# them both from charging.  If the away vehicle is not currently
	// 		# charging, I'm not sure if this would prevent it from charging
	// 		# when next plugged in.
	// 		queue_background_task({'cmd':'charge', 'charge':False})

	// 	elif(self.lastAmpsOffered >= 5.0 and self.reportedAmpsActual < 2.0
	// 			and self.reportedState != 0x02
	// 	):
	// 		# Car is not charging and is not reporting an error state, so
	// 		# try starting charge via car api.
	// 		queue_background_task({'cmd':'charge', 'charge':True})

	// 	elif(self.reportedAmpsActual > 4.0):
	// 		# At least one plugged in car is successfully charging. We don't
	// 		# know which car it is, so we must set
	// 		# vehicle.stopAskingToStartCharging = False on all vehicles such
	// 		# that if any vehicle is not charging without us calling
	// 		# car_api_charge(False), we'll try to start it charging again at
	// 		# least once. This probably isn't necessary but might prevent
	// 		# some unexpected case from never starting a charge. It also
	// 		# seems less confusing to see in the output that we always try
	// 		# to start API charging after the car stops taking a charge.
	// 		for vehicle in carApiVehicles:
	// 			vehicle.stopAskingToStartCharging = False

	// send_msg(bytearray(b'\xFB\xE0') + fakeTWCID + bytearray(self.TWCID)
	// 			+ bytearray(self.masterHeartbeatData))

	msg := bytes.NewBuffer([]byte{0xFB, 0xE0})
	msg.Write(fakeTWCID)
	msg.Write(h.twcID)
	msg.Write(h.masterHeartbeatData)

	return master.send(msg.Bytes())
}

func (h *Slave) receiveSlaveHeartbeat(heartbeatData []byte) error {
	return nil
}
