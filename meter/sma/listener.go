package sma

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/andig/evcc/api"
)

const (
	multicastAddr    = "239.12.255.254:9522"
	udpBufferSize    = 8192
	obisCodeLength   = 4
	ObisImportPower  = "1:1.4.0" // Wirkleistung (W)
	ObisImportEnergy = "1:1.8.0" // Wirkarbeit (Ws) +
	ObisExportPower  = "1:2.4.0" // Wirkleistung (W)
	ObisExportEnergy = "1:2.8.0" // Wirkarbeit (Ws) −
)

// obisCodeProp defines the properties needed to parse the SMA multicast telegram values
type obisCodeProp struct {
	length int     // data size in bytes of the return value
	factor float64 // the factor to multiply the value by to get the proper value in the given unit
}

// list of Obis codes and their properties as defined in the SMA EMETER-Protokoll-TI-de-10.pdf document
var knownObisCodes = map[string]obisCodeProp{
	// Overal sums
	ObisImportPower: {4, 0.1}, ObisImportEnergy: {8, 1}, // Wirkleistung (W)/-arbeit (Ws) +
	ObisExportPower: {4, 0.1}, ObisExportEnergy: {8, 1}, // Wirkleistung (W)/-arbeit (Ws) −
	"1:3.4.0": {4, 0.1}, "1:3.8.0": {8, 1}, // Blindleistung (W)/-arbeit (Ws) +
	"1:4.4.0": {4, 0.1}, "1:4.8.0": {8, 1}, // Blindleistung (W)/-arbeit (Ws) −
	"1:9.4.0": {4, 0.1}, "1:9.8.0": {8, 1}, // Scheinleistung (W)/-arbeit (Ws) +
	"1:10.4.0": {4, 0.1}, "1:10.8.0": {8, 1}, // Scheinleistung (W)/-arbeit (Ws) −
	"1:13.4.0": {4, 0.001}, // Leistungsfaktor (Φ)
	// Phase 1: {
	"1:21.4.0": {4, 0.1}, "1:21.8.0": {8, 1}, // Wirkleistung (W)/-arbeit (Ws) +
	"1:22.4.0": {4, 0.1}, "1:22.8.0": {8, 1}, // Wirkleistung (W)/-arbeit (Ws) −
	"1:23.4.0": {4, 0.1}, "1:23.8.0": {8, 1}, // Blindleistung (W)/-arbeit (Ws) +
	"1:24.4.0": {4, 0.1}, "1:24.8.0": {8, 1}, // Blindleistung (W)/-arbeit (Ws) −
	"1:29.4.0": {4, 0.1}, "1:29.8.0": {8, 1}, // Scheinleistung (W)/-arbeit (Ws) +
	"1:30.4.0": {4, 0.1}, "1:30.8.0": {8, 1}, // Scheinleistung (W)/-arbeit (Ws) −
	"1:31.4.0": {4, 0.001}, // Strom (A)
	"1:32.4.0": {4, 0.001}, // Spannung (V
	// Phase 2: {
	"1:41.4.0": {4, 0.1}, "1:41.8.0": {8, 1}, // Wirkleistung (W)/-arbeit (Ws) +
	"1:42.4.0": {4, 0.1}, "1:42.8.0": {8, 1}, // Wirkleistung (W)/-arbeit (Ws) −
	"1:43.4.0": {4, 0.1}, "1:43.8.0": {8, 1}, // Blindleistung (W)/-arbeit (Ws) +
	"1:44.4.0": {4, 0.1}, "1:44.8.0": {8, 1}, // Blindleistung (W)/-arbeit (Ws) −
	"1:49.4.0": {4, 0.1}, "1:49.8.0": {8, 1}, // Scheinleistung (W)/-arbeit (Ws) +
	"1:50.4.0": {4, 0.1}, "1:50.8.0": {8, 1}, // Scheinleistung (W)/-arbeit (Ws) −
	"1:51.4.0": {4, 0.001}, // Strom (A)
	"1:52.4.0": {4, 0.001}, // Spannung (V)
	// Phase 3: {
	"1:61.4.0": {4, 0.1}, "1:61.8.0": {8, 1}, // Wirkleistung (W)/-arbeit (Ws) +
	"1:62.4.0": {4, 0.1}, "1:62.8.0": {8, 1}, // Wirkleistung (W)/-arbeit (Ws) −
	"1:63.4.0": {4, 0.1}, "1:63.8.0": {8, 1}, // Blindleistung (W)/-arbeit (Ws) +
	"1:64.4.0": {4, 0.1}, "1:64.8.0": {8, 1}, // Blindleistung (W)/-arbeit (Ws) −
	"1:69.4.0": {4, 0.1}, "1:69.8.0": {8, 1}, // Scheinleistung (W)/-arbeit (Ws) +
	"1:70.4.0": {4, 0.1}, "1:70.8.0": {8, 1}, // Scheinleistung (W)/-arbeit (Ws) −
	"1:71.4.0": {4, 0.001}, // Strom (A)
	"1:72.4.0": {4, 0.001}, // Spannung (V)
	// Others
	"144:0.0.0": {4, 1}, // SW Version
}

var Instance *Listener

// TelegramData defines the data structure of a SMA multicast data package
type TelegramData struct {
	Addr   string
	Serial string
	Data   map[string]float64
}

// Listener for receiving SMA multicast data packages
type Listener struct {
	mux     sync.Mutex
	log     *api.Logger
	conn    *net.UDPConn
	clients map[string]chan<- TelegramData
}

// New creates a Listener
func New(log *api.Logger, addr string) *Listener {
	// Parse the string address
	laddr, err := net.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		log.FATAL.Fatalf("error resolving udp address: %s", err)
	}

	// Open up a connection
	conn, err := net.ListenMulticastUDP("udp4", nil, laddr)
	if err != nil {
		log.FATAL.Fatalf("error opening connecting: %s", err)
	}

	if err := conn.SetReadBuffer(udpBufferSize); err != nil {
		log.FATAL.Fatalf("error setting read buffer: %s", err)
	}

	l := &Listener{
		log:  log,
		conn: conn,
	}

	go l.listen()

	return l
}

// processUDPData converts a SMA Multicast data package into TelegramData
func (l *Listener) processUDPData(src *net.UDPAddr, buffer []byte) (TelegramData, error) {
	numBytes := len(buffer)

	if numBytes < 29 {
		return TelegramData{}, errors.New("received data package is too small")
	}

	obisCodeValues := make(map[string]float64)

	// yes this doesn't look nice, but keeping it until we found a better way to parse the data
	// read obis code values, start at position 28, after initial static stuff
	for i := 28; i < numBytes; i++ {
		if i+obisCodeLength > numBytes-1 {
			break
		}

		// create the string notation of the potential obis code
		b := buffer[i : i+obisCodeLength]

		// Spec says value should be 1, but reading contains 0
		b3 := b[0]
		if b3 == 0 {
			b3 = 1
		}

		code := fmt.Sprintf("%d:%d.%d.%d", b3, b[1], b[2], b[3])

		element, ok := knownObisCodes[code]
		if !ok {
			continue
		}

		dataIndex := i + obisCodeLength

		var value float64
		switch element.length {
		case 4:
			value = float64(binary.BigEndian.Uint32(buffer[dataIndex : dataIndex+element.length+1]))
		case 8:
			value = float64(binary.BigEndian.Uint64(buffer[dataIndex : dataIndex+element.length+1]))
		}

		obisCodeValues[code] = value * element.factor

		i = dataIndex + element.length - 1
	}

	serial := strconv.FormatUint(uint64(binary.BigEndian.Uint32(buffer[20:24])), 10)

	msg := TelegramData{
		Addr:   src.IP.String(),
		Serial: serial,
		Data:   obisCodeValues,
	}

	return msg, nil
}

// listen for Multicast data packages
func (l *Listener) listen() {
	buffer := make([]byte, udpBufferSize)
	// Loop forever reading
	for {
		numBytes, src, err := l.conn.ReadFromUDP(buffer)
		if err != nil {
			l.log.WARN.Printf("readfromudp failed: %s", err)
			continue
		}

		if msg, err := l.processUDPData(src, buffer[:numBytes-1]); err == nil {
			l.send(msg)
		}
	}
}

// Subscribe adds a client address and message channel
func (l *Listener) Subscribe(addr string, c chan<- TelegramData) {
	l.mux.Lock()
	defer l.mux.Unlock()

	if l.clients == nil {
		l.clients = make(map[string]chan<- TelegramData)
	}

	l.clients[addr] = c
}

func (l *Listener) send(msg TelegramData) {
	l.mux.Lock()
	defer l.mux.Unlock()

	for addr, client := range l.clients {
		if addr == msg.Addr {
			select {
			case client <- msg:
			default:
				l.log.TRACE.Println("listener: recv blocked")
			}
			break
		}
	}
}
