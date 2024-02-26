package goodwe

import (
	"encoding/binary"
	"net"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/exp/maps"
)

var (
	instance *Server
	mu       sync.RWMutex
)

func Instance(log *util.Logger) (*Server, error) {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		return instance, nil
	}

	instance = &Server{
		log:       log,
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
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Second
	bo.MaxElapsedTime = 10 * time.Second

	for {
		mu.RLock()
		ips := maps.Keys(m.inverters)
		mu.RUnlock()

		for _, ip := range ips {
			if err := backoff.Retry(func() error {
				addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ip, "8899"))
				if err == nil {
					_, err = m.conn.WriteToUDP([]byte{0xF7, 0x03, 0x89, 0x1C, 0x00, 0x7D, 0x7A, 0xE7}, addr)
				}
				if err == nil {
					time.Sleep(5 * time.Second)
					_, err = m.conn.WriteToUDP([]byte{0xF7, 0x03, 0x90, 0x88, 0x00, 0x0D, 0x3D, 0xB3}, addr)
				}
				return err
			}, bo); err != nil {
				m.log.ERROR.Println(err)
			}
		}
	}
}

func (m *Server) listen() {
	for {
		buf := make([]byte, 1024)
		n, addr, err := m.conn.ReadFromUDP(buf)
		if err != nil {
			m.log.ERROR.Println(err)
			continue
		}
		m.log.TRACE.Printf("recv from %s: % X", addr, buf[:n])

		ip := addr.IP.String()
		monitor := m.GetInverter(ip)
		if monitor == nil {
			m.log.ERROR.Println("unknown inverter:", ip)
			continue
		}

		monitor.SetFunc(func(inverter Inverter) Inverter {
			if buf[4] == 250 {
				ui := func(u, i int16) float64 {
					return float64(int16(binary.BigEndian.Uint16(buf[u:]))) *
						float64(int16(binary.BigEndian.Uint16(buf[i:]))) / 100
				}
				inverter.PvPower = ui(11, 13) + ui(19, 21)
				inverter.BatteryPower = ui(165, 167)
				inverter.NetPower = -float64(int32(binary.BigEndian.Uint32(buf[83:])))
			}

			if buf[4] == 26 {
				inverter.Soc = float64(buf[20])
			}

			return inverter
		})
	}
}
