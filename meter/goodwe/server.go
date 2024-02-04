package goodwe

import (
	"encoding/binary"
	"net"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
)

var (
	instance *Server
	mu       sync.RWMutex
)

func Instance() (*Server, error) {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		return instance, nil
	}

	instance = &Server{
		inverters: make(map[string]*util.Monitor[Inverter]),
	}

	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:8899")
	if err != nil {
		return nil, err
	}

	instance.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	go instance.listen()
	go instance.readData()

	return instance, err
}

func (m *Server) AddInverter(ip string, timeout time.Duration) *util.Monitor[Inverter] {
	mu.Lock()
	defer mu.Unlock()
	monitor := util.NewMonitor[Inverter](timeout)
	m.inverters[ip] = monitor
	return monitor
}

func (m *Server) GetInverter(ip string) *util.Monitor[Inverter] {
	mu.RLock()
	defer mu.RUnlock()
	return m.inverters[ip]
}

func (m *Server) readData() {
	mu.RLock()
	defer mu.RUnlock()

	for ip := range m.inverters {
		addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ip, "8899"))
		if err != nil {
			return
		}
		if _, err := m.conn.WriteToUDP([]byte{0xF7, 0x03, 0x89, 0x1C, 0x00, 0x7D, 0x7A, 0xE7}, addr); err != nil {
			return
		}
		time.Sleep(5 * time.Second)
		if _, err := m.conn.WriteToUDP([]byte{0xF7, 0x03, 0x90, 0x88, 0x00, 0x0D, 0x3D, 0xB3}, addr); err != nil {
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

		monitor := m.GetInverter(addr.IP.String())
		if monitor == nil {
			continue
		}

		monitor.SetFunc(func(inverter Inverter) Inverter {
			if buf[4] == 250 {
				vPv1 := float64(int16(binary.BigEndian.Uint16(buf[11:]))) * 0.1
				vPv2 := float64(int16(binary.BigEndian.Uint16(buf[19:]))) * 0.1
				iPv1 := float64(int16(binary.BigEndian.Uint16(buf[13:]))) * 0.1
				iPv2 := float64(int16(binary.BigEndian.Uint16(buf[21:]))) * 0.1
				iBatt := float64(int16(binary.BigEndian.Uint16(buf[167:]))) * 0.1
				vBatt := float64(int16(binary.BigEndian.Uint16(buf[165:]))) * 0.1

				pvPower := vPv1*iPv1 + vPv2*iPv2

				inverter.PvPower = pvPower
				inverter.BatteryPower = vBatt * iBatt
				inverter.NetPower = -float64(int32(binary.BigEndian.Uint32(buf[83:])))
			}

			if buf[4] == 26 {
				inverter.Soc = float64(buf[20])
			}

			return inverter
		})
	}
}
