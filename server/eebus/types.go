package eebus

import "github.com/evcc-io/evcc/util"

const (
	BrandName string = "EVCC"
	Model     string = "HEMS"
)

// used as common name in cert generation
var DeviceCode = util.Getenv("EEBUS_DEVICE_CODE", "EVCC_HEMS_01")

type Config struct {
	URI         string
	ShipID      string
	Interfaces  []string
	Certificate struct {
		Public, Private string
	}
}

// Configured returns true if the EEbus server is configured
func (c Config) Configured() bool {
	return len(c.Certificate.Public) > 0 && len(c.Certificate.Private) > 0
}
