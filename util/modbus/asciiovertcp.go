package modbus

import (
	"time"

	gridx "github.com/grid-x/modbus"
	"github.com/volkszaehler/mbmd/meters"
)

// ASCIIOverTCP is an RTU encoder over a TCP modbus connection
type ASCIIOverTCP struct {
	address string
	Client  gridx.Client
	Handler *gridx.ASCIIOverTCPClientHandler
	prevID  uint8
}

// NewASCIIOverTCPClientHandler creates a TCP modbus handler
func NewASCIIOverTCPClientHandler(device string) *gridx.ASCIIOverTCPClientHandler {
	handler := gridx.NewASCIIOverTCPClientHandler(device)

	// set default timings
	handler.Timeout = 1 * time.Second
	handler.ProtocolRecoveryTimeout = 10 * time.Second // not used
	handler.LinkRecoveryTimeout = 15 * time.Second     // not used

	return handler
}

// NewASCIIOverTCP creates a TCP modbus client
func NewASCIIOverTCP(address string) meters.Connection {
	handler := NewASCIIOverTCPClientHandler(address)
	client := gridx.NewClient(handler)

	b := &ASCIIOverTCP{
		address: address,
		Client:  client,
		Handler: handler,
	}

	return b
}

// String returns the bus connection address (TCP)
func (b *ASCIIOverTCP) String() string {
	return b.address
}

// ModbusClient returns the TCP modbus client
func (b *ASCIIOverTCP) ModbusClient() gridx.Client {
	return b.Client
}

// Logger sets a logging instance for physical bus operations
func (b *ASCIIOverTCP) Logger(l meters.Logger) {
	b.Handler.Logger = l
}

// Slave sets the modbus device id for the following operations
func (b *ASCIIOverTCP) Slave(deviceID uint8) {
	// Some devices like SDM need to have a little pause between querying different device ids
	if b.prevID != 0 && deviceID != b.prevID {
		time.Sleep(time.Duration(100) * time.Millisecond)
		b.prevID = deviceID
	}

	b.Handler.SetSlave(deviceID)
}

// Timeout sets the modbus timeout
func (b *ASCIIOverTCP) Timeout(timeout time.Duration) time.Duration {
	t := b.Handler.Timeout
	b.Handler.Timeout = timeout
	return t
}

// Close closes the modbus connection.
// This forces the modbus client to reopen the connection before the next bus operations.
func (b *ASCIIOverTCP) Close() {
	b.Handler.Close()
}
