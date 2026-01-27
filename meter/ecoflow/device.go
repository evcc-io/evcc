package ecoflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const defaultURI = "https://api-e.ecoflow.com"

// Device is the base type for EcoFlow devices
type Device struct {
	*request.Helper
	uri   string
	sn    string
	usage string
	cache time.Duration
}

// NewDevice creates a new EcoFlow device from config map
func NewDevice(other map[string]any, deviceType string) (*Device, error) {
	var cc config
	if err := cc.decode(other); err != nil {
		return nil, err
	}
	
	// Validation
	if cc.SN == "" || cc.AccessKey == "" || cc.SecretKey == "" {
		return nil, fmt.Errorf("%s: missing sn, accessKey or secretKey", deviceType)
	}
	
	uri := cc.URI
	if uri == "" {
		uri = defaultURI
	}

	log := util.NewLogger(deviceType).Redact(cc.AccessKey, cc.SecretKey)

	d := &Device{
		Helper: request.NewHelper(log),
		uri:    strings.TrimSuffix(uri, "/"),
		sn:     cc.SN,
		usage:  strings.ToLower(cc.Usage),
		cache:  cc.Cache,
	}

	d.Client.Transport = NewAuthTransport(d.Client.Transport, cc.AccessKey, cc.SecretKey)

	return d, nil
}

// quotaURL returns the API URL for device quota
func (d *Device) quotaURL() string {
	return fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", d.uri, d.sn)
}
