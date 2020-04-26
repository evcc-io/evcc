package smameter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/andig/evcc/api"
)

const (
	multicast_addr = "239.12.255.254:9522"
	udpBufferSize  = 8192
	obisCodeLength = 4
)

// obisCode defines the properties needed to parse the SMA multicast telegram values
type obisCode struct {
	obisCode string  // the obis code of the returned data type
	length   int     // data size in bytes of the return value
	factor   float64 // the factor to multiply the value by to get the proper value in the given unit
	value    float64 // the actual provided return value
}

// list of Obis codes and their properties as defined in the SMA EMETER-Protokoll-TI-de-10.pdf document
var knownObisCodes = []obisCode{
	// Overal sums
	{"1:1.4.0", 4, 0.1, 0}, {"1:1.8.0", 8, 1, 0}, // Wirkleistung (W)/-arbeit (Ws) +
	{"1:2.4.0", 4, 0.1, 0}, {"1:2.8.0", 8, 1, 0}, // Wirkleistung (W)/-arbeit (Ws) −
	{"1:3.4.0", 4, 0.1, 0}, {"1:3.8.0", 8, 1, 0}, // Blindleistung (W)/-arbeit (Ws) +
	{"1:4.4.0", 4, 0.1, 0}, {"1:4.8.0", 8, 1, 0}, // Blindleistung (W)/-arbeit (Ws) −
	{"1:9.4.0", 4, 0.1, 0}, {"1:9.8.0", 8, 1, 0}, // Scheinleistung (W)/-arbeit (Ws) +
	{"1:10.4.0", 4, 0.1, 0}, {"1:10.8.0", 8, 1, 0}, // Scheinleistung (W)/-arbeit (Ws) −
	{"1:13.4.0", 4, 0.001, 0}, // Leistungsfaktor (Φ)
	// Phase 1
	{"1:21.4.0", 4, 0.1, 0}, {"1:21.8.0", 8, 1, 0}, // Wirkleistung (W)/-arbeit (Ws) +
	{"1:22.4.0", 4, 0.1, 0}, {"1:22.8.0", 8, 1, 0}, // Wirkleistung (W)/-arbeit (Ws) −
	{"1:23.4.0", 4, 0.1, 0}, {"1:23.8.0", 8, 1, 0}, // Blindleistung (W)/-arbeit (Ws) +
	{"1:24.4.0", 4, 0.1, 0}, {"1:24.8.0", 8, 1, 0}, // Blindleistung (W)/-arbeit (Ws) −
	{"1:29.4.0", 4, 0.1, 0}, {"1:29.8.0", 8, 1, 0}, // Scheinleistung (W)/-arbeit (Ws) +
	{"1:30.4.0", 4, 0.1, 0}, {"1:30.8.0", 8, 1, 0}, // Scheinleistung (W)/-arbeit (Ws) −
	{"1:31.4.0", 4, 0.001, 0}, // Strom (A)
	{"1:32.4.0", 4, 0.001, 0}, // Spannung (V
	// Phase 2
	{"1:41.4.0", 4, 0.1, 0}, {"1:41.8.0", 8, 1, 0}, // Wirkleistung (W)/-arbeit (Ws) +
	{"1:42.4.0", 4, 0.1, 0}, {"1:42.8.0", 8, 1, 0}, // Wirkleistung (W)/-arbeit (Ws) −
	{"1:43.4.0", 4, 0.1, 0}, {"1:43.8.0", 8, 1, 0}, // Blindleistung (W)/-arbeit (Ws) +
	{"1:44.4.0", 4, 0.1, 0}, {"1:44.8.0", 8, 1, 0}, // Blindleistung (W)/-arbeit (Ws) −
	{"1:49.4.0", 4, 0.1, 0}, {"1:49.8.0", 8, 1, 0}, // Scheinleistung (W)/-arbeit (Ws) +
	{"1:50.4.0", 4, 0.1, 0}, {"1:50.8.0", 8, 1, 0}, // Scheinleistung (W)/-arbeit (Ws) −
	{"1:51.4.0", 4, 0.001, 0}, // Strom (A)
	{"1:52.4.0", 4, 0.001, 0}, // Spannung (V
	// Phase 3
	{"1:61.4.0", 4, 0.1, 0}, {"1:61.8.0", 8, 1, 0}, // Wirkleistung (W)/-arbeit (Ws) +
	{"1:62.4.0", 4, 0.1, 0}, {"1:62.8.0", 8, 1, 0}, // Wirkleistung (W)/-arbeit (Ws) −
	{"1:63.4.0", 4, 0.1, 0}, {"1:63.8.0", 8, 1, 0}, // Blindleistung (W)/-arbeit (Ws) +
	{"1:64.4.0", 4, 0.1, 0}, {"1:64.8.0", 8, 1, 0}, // Blindleistung (W)/-arbeit (Ws) −
	{"1:69.4.0", 4, 0.1, 0}, {"1:69.8.0", 8, 1, 0}, // Scheinleistung (W)/-arbeit (Ws) +
	{"1:70.4.0", 4, 0.1, 0}, {"1:70.8.0", 8, 1, 0}, // Scheinleistung (W)/-arbeit (Ws) −
	{"1:71.4.0", 4, 0.001, 0}, // Strom (A)
	{"1:72.4.0", 4, 0.001, 0}, // Spannung (V)
	// Others
	{"144:0.0.0", 4, 1, 0}, // SW Version
}

var Instance *Listener

// SmaObisCodeValue contains the value of a specific OBIS code
type SmaObisCodeValue struct {
	ObisCode string
	Value    float64
}

// SmaTelegramData defines the data structure of a SMA multicast data package
type SmaTelegramData struct {
	Addr   string
	Serial string
	Data   []SmaObisCodeValue
}

// Listener for receiving SMA multicast data packages
type Listener struct {
	mux     sync.Mutex
	log     *api.Logger
	conn    *net.UDPConn
	clients map[string]chan<- SmaTelegramData
}

// New creates a Listener
func New(log *api.Logger, addr string) *Listener {
	// Parse the string address
	laddr, err := net.ResolveUDPAddr("udp4", multicast_addr)
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

// processUDPData converts a SMA Multicast data package into SmaTelegramData
func (l *Listener) processUDPData(numBytes int, src *net.UDPAddr, buffer []byte) (SmaTelegramData, error) {
	// read serial number at index 20
	serial := strconv.FormatUint(uint64(readSerial(buffer[20:24])), 10)

	var obisCodeValues []SmaObisCodeValue

	// read obis code values, start at position 28, after initial static stuff
	for i := 28; i < numBytes; i++ {
		if i+obisCodeLength > numBytes-1 {
			break
		}
		if obisCodeElement, err := getObisCodeElement(buffer[i : i+obisCodeLength]); err == nil {
			dataIndex := i + obisCodeLength

			obisCodeValue := readValue(buffer[dataIndex:dataIndex+obisCodeElement.length+1], obisCodeElement.length)
			obisCodeElement.value = obisCodeValue * obisCodeElement.factor

			smaObisCodeValue := SmaObisCodeValue{
				ObisCode: obisCodeElement.obisCode,
				Value:    obisCodeValue * obisCodeElement.factor,
			}

			obisCodeValues = append(obisCodeValues, smaObisCodeValue)

			i = dataIndex + obisCodeElement.length - 1
		}
	}

	msg := SmaTelegramData{
		Addr:   src.IP.String(),
		Serial: serial,
		Data:   obisCodeValues,
	}

	return msg, nil
}

// listen for Multicast data packages
func (l *Listener) listen() {
	// Loop forever reading
	for {
		buffer := make([]byte, udpBufferSize)
		numBytes, src, err := l.conn.ReadFromUDP(buffer)
		if err != nil {
			l.log.WARN.Fatalf("readfromudp failed: %s", err)
			continue
		}

		if msg, err := l.processUDPData(numBytes, src, buffer); err != nil {
			l.send(msg)
		}
	}

}

// Subscribe adds a client address and message channel
func (l *Listener) Subscribe(addr string, c chan<- SmaTelegramData) {
	l.mux.Lock()
	defer l.mux.Unlock()

	if l.clients == nil {
		l.clients = make(map[string]chan<- SmaTelegramData)
	}

	l.clients[addr] = c
}

func (l *Listener) send(msg SmaTelegramData) {
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

// getObisCodeElement parses the OBIS code from a set of bytes
func getObisCodeElement(value []byte) (obisCode, error) {
	b0 := value[3]
	b1 := value[2]
	b2 := value[1]
	b3 := value[0]

	// Spec says value should be 1, but reading contains 0
	if b3 == 0 {
		b3 = 1
	}

	var obisCodeElement obisCode

	err := fmt.Errorf("no obis code found")

	for _, code := range knownObisCodes {
		obisCode := fmt.Sprintf("%d:%d.%d.%d", b3, b2, b1, b0)
		if code.obisCode == obisCode {
			obisCodeElement = code
			err = nil
			break
		}
	}

	return obisCodeElement, err
}

// readSerial reads the serial number from the binary data
func readSerial(b []byte) uint32 {
	reader := bytes.NewReader(b)

	var i uint32

	if err := binary.Read(reader, binary.BigEndian, &i); err != nil {
		return 0
	}

	return i
}

// readValue gets the value from the binary data according to the provided data length in bytes
func readValue(b []byte, length int) float64 {
	reader := bytes.NewReader(b)

	switch length {
	case 4:
		var i uint32

		if err := binary.Read(reader, binary.BigEndian, &i); err != nil {
			return 0
		}

		return float64(i)
	case 8:
		var i uint64

		if err := binary.Read(reader, binary.BigEndian, &i); err != nil {
			return 0
		}

		return float64(i)
	}

	return 0
}
