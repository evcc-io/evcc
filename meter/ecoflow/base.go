package ecoflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Device is the base type for EcoFlow devices (Stream and PowerStream)
// It encapsulates the common configuration and HTTP handling logic
type Device struct {
	*request.Helper
	log       *util.Logger
	uri       string
	sn        string
	accessKey string
	secretKey string
	usage     Usage
	cache     time.Duration
}

// NewDevice creates a new EcoFlow device with shared configuration
func NewDevice(deviceName, uri, sn, accessKey, secretKey string, usage Usage, cache time.Duration) (*Device, error) {
	if err := ValidateConfig(uri, sn, accessKey, secretKey, deviceName); err != nil {
		return nil, err
	}

	log := util.NewLogger(deviceName).Redact(accessKey, secretKey)

	device := &Device{
		Helper:    request.NewHelper(log),
		log:       log,
		uri:       strings.TrimSuffix(uri, "/"),
		sn:        sn,
		accessKey: accessKey,
		secretKey: secretKey,
		usage:     usage,
		cache:     cache,
	}

	// Set authorization header using custom transport with HMAC-SHA256 signature
	device.Client.Transport = NewEcoFlowAuthTransport(device.Client.Transport, accessKey, secretKey)

	return device, nil
}

// GetURI returns the device URI
func (d *Device) GetURI() string {
	return d.uri
}

// GetSN returns the device serial number
func (d *Device) GetSN() string {
	return d.sn
}

// GetUsage returns the device usage type
func (d *Device) GetUsage() Usage {
	return d.usage
}

// GetCache returns the cache duration
func (d *Device) GetCache() time.Duration {
	return d.cache
}

// GetClient returns the HTTP client
func (d *Device) GetClient() *request.Helper {
	return d.Helper
}
