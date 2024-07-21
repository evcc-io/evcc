package eebus

const (
	BrandName  string = "EVCC"
	Model      string = "HEMS"
	DeviceCode string = "EVCC_HEMS_01" // used as common name in cert generation
)

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
