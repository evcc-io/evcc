package meter

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Server struct {
	conn      *net.UDPConn
	inverters map[string]Inverter
}

type Inverter struct {
	IP           string
	pvPower      float64
	netPower     float64
	batteryPower float64
	soc          float64
}

var server *Server

type GoodWeWiFiMeter struct {
	usage string
	URI   string
}

func init() {
	registry.Add("goodwe-wifi", NewGoodWeWifiFromConfig)
}

func NewGoodWeWifiFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		capacity   `mapstructure:",squash"`
		URI, Usage string
		Cache      time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewGoodWeWiFiMeter(cc.URI, cc.Usage)
}

func NewGoodWeWiFiMeter(uri string, usage string) (api.Meter, error) {
	meter := &GoodWeWiFiMeter{
		usage: usage,
		URI:   uri,
	}

	server, err := NewServer()
	if err != nil {
		return nil, err
	}
	server.addInverter(uri)

	return meter, nil
}

func (m *GoodWeWiFiMeter) CurrentPower() (float64, error) {
	switch m.usage {
	case "grid":
		return server.inverters[m.URI].netPower, nil
	case "pv":
		return server.inverters[m.URI].pvPower, nil
	case "battery":
		return server.inverters[m.URI].batteryPower, nil
	}
	return server.inverters[m.URI].pvPower, api.ErrNotAvailable
}

func NewServer() (*Server, error) {
	if server == nil {
		server = &Server{
			inverters: make(map[string]Inverter),
		}
		addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:8899")
		if err != nil {
			return nil, err
		}

		server.conn, err = net.ListenUDP("udp", addr)

		if err != nil {
			return nil, err
		}

		go server.listen()

		go server.readData()

		return server, err
	} else {
		return server, nil
	}
}

func (m *Server) addInverter(ip string) {
	server.inverters[ip] = Inverter{IP: ip}
}

func (m *Server) readData() {
	for _, inverter := range server.inverters {
		addr, err := net.ResolveUDPAddr("udp", inverter.IP+":8899")

		server.conn.WriteToUDP([]byte{0xF7, 0x03, 0x89, 0x1C, 0x00, 0x7D, 0x7A, 0xE7}, addr)

		time.Sleep(5 * time.Second)

		server.conn.WriteToUDP([]byte{0xF7, 0x03, 0x90, 0x88, 0x00, 0x0D, 0x3D, 0xB3}, addr)

		if err != nil {
			return
		}
	}
	m.readData()
}

func (m *Server) listen() {
	for {
		buf := make([]byte, 1024)
		_, addr, err := m.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		ip := addr.IP.String()

		if buf[4] == 250 {
			vPv1 := float64(int16(binary.BigEndian.Uint16(buf[11:]))) * 0.1
			vPv2 := float64(int16(binary.BigEndian.Uint16(buf[19:]))) * 0.1
			iPv1 := float64(int16(binary.BigEndian.Uint16(buf[13:]))) * 0.1
			iPv2 := float64(int16(binary.BigEndian.Uint16(buf[21:]))) * 0.1
			iBatt := float64(int16(binary.BigEndian.Uint16(buf[167:]))) * 0.1
			vBatt := float64(int16(binary.BigEndian.Uint16(buf[165:]))) * 0.1

			pvPower := vPv1*iPv1 + vPv2*iPv2

			inverter := server.inverters[ip]
			inverter.pvPower = pvPower
			inverter.batteryPower = vBatt * iBatt
			inverter.netPower = float64(int32(binary.BigEndian.Uint32(buf[83:])))

			server.inverters[ip] = inverter
		}

		if buf[4] == 26 {
			inverter := server.inverters[ip]
			inverter.soc = float64(int16(binary.BigEndian.Uint16(buf[19:])))
		}
	}
}
