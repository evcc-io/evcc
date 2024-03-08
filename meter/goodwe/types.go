package goodwe

import (
	"net"

	"github.com/evcc-io/evcc/util"
)

type Server struct {
	log       *util.Logger
	conn      *net.UDPConn
	inverters map[string]*util.Monitor[Inverter]
}

type Inverter struct {
	PvPower      float64
	NetPower     float64
	BatteryPower float64
	Soc          float64
}
