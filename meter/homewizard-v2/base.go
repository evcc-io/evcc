package v2

import (
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// DeviceType identifies the type of HomeWizard device
type DeviceType string

const (
	DeviceTypeP1Meter  DeviceType = "p1meter"
	DeviceTypeKWHMeter DeviceType = "kwhmeter"
	DeviceTypeBattery  DeviceType = "battery"
)

// deviceBase contains common functionality for all HomeWizard devices
type deviceBase struct {
	*request.Helper
	deviceType DeviceType
	host       string
	token      string
	log        *util.Logger
	conn       *Connection
	timeout    time.Duration
}

// newDeviceBase creates a new base device with common fields
func newDeviceBase(deviceType DeviceType, host, token string, timeout time.Duration) *deviceBase {
	log := util.NewLogger("homewizard-v2").Redact(token)

	d := &deviceBase{
		Helper:     request.NewHelper(log),
		deviceType: deviceType,
		host:       host,
		token:      token,
		log:        log,
		timeout:    timeout,
	}

	// Use insecure HTTPS transport for self-signed certificates
	d.Client.Transport = transport.Insecure()

	// Set timeout for HTTP requests
	d.Client.Timeout = 10 * time.Second

	return d
}

// Type returns the device type
func (d *deviceBase) Type() DeviceType {
	return d.deviceType
}

// Host returns the device hostname/IP
func (d *deviceBase) Host() string {
	return d.host
}

// Start initiates the WebSocket connection
func (d *deviceBase) Start(errC chan error) {
	d.conn.Start(errC)
}

// Stop gracefully closes the connection
func (d *deviceBase) Stop() {
	d.conn.Stop()
}
