package twc2

import (
	"bytes"
	"encoding/binary"

	"github.com/andig/evcc/util"
)

var (
	slaveHeartbeatData = []byte{0x01, 0x0F, 0xA0, 0x0F, 0xA0, 0x00, 0x00, 0x00, 0x00}
)

// Slave is a TWC slave instance
type Slave struct {
	log                 *util.Logger
	twcID               []byte
	protocolVersion     int
	minAmpsTWCSupports  int
	wiringMaxAmps       int
	masterHeartbeatData []byte
	state               byte
	amps                int
}

// NewSlave creates a new slave instance
func NewSlave(log *util.Logger, slaveID uint16, maxAmps int) *Slave {
	log.DEBUG.Printf("new slave: %4X maxAmps: %d", slaveID, maxAmps)

	h := &Slave{
		log:                 log,
		twcID:               make([]byte, 2),
		protocolVersion:     1,
		minAmpsTWCSupports:  6,
		wiringMaxAmps:       maxAmps,
		masterHeartbeatData: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	binary.BigEndian.PutUint16(h.twcID, slaveID)

	return h
}

func (h *Slave) sendMasterHeartbeat() error {
	h.log.TRACE.Println("sendMasterHeartbeat")

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

func (h *Slave) receiveHeartbeat(payload SlaveHeartbeatPayload) error {
	h.amps = int(payload.AmpsActual / 100)
	h.state = payload.State

	h.log.DEBUG.Printf("heartbeat state: %d amps: %d", h.state, h.amps)

	return nil
}
