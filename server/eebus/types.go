package eebus

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/machine"
)

const (
	BrandName string = "EVCC"
	Model     string = "HEMS"
)

// used as common name in cert generation
var DeviceCode = util.Getenv("EEBUS_DEVICE_CODE", "EVCC_HEMS_01")

type Certificate struct {
	Public, Private string
}

type Config struct {
	URI_        string `mapstructure:"uri"` // TODO deprecated
	Port        int
	ShipID      string
	Interfaces  []string
	Certificate Certificate
}

// IsConfigured returns true if the EEbus server is configured
func (c Config) IsConfigured() bool {
	return len(c.Certificate.Public) > 0 && len(c.Certificate.Private) > 0
}

func createShipID() string {
	protectedID := machine.ProtectedID("evcc-eebus")
	return fmt.Sprintf("%s-%0x", "EVCC", protectedID[:8])
}

func DefaultConfig() (*Config, error) {
	cert, err := CreateCertificate()
	if err != nil {
		return nil, err
	}

	public, private, err := GetX509KeyPair(cert)
	if err != nil {
		return nil, err
	}

	res := Config{
		Port:   4712,
		ShipID: createShipID(),
		Certificate: Certificate{
			Public:  public,
			Private: private,
		},
	}

	return &res, nil
}
