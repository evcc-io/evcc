package goodwe

import (
	"net"
)

type Server struct {
	conn      *net.UDPConn
	inverters map[string]Inverter
}

type Inverter struct {
	IP           string
	PvPower      float64
	NetPower     float64
	BatteryPower float64
	Soc          float64
}
