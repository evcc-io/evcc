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
	Public  string `json:"public"`
	Private string `json:"private"`
}

type Config struct {
	URI_        string      `mapstructure:"uri" json:"uri"` // TODO deprecated
	Port        int         `json:"port"`
	ShipID      string      `json:"shipid"`
	Interfaces  []string    `json:"interfaces,omitempty"`
	Certificate Certificate `json:"certificate"`
}

// IsConfigured returns true if the EEbus server is configured
func (c Config) IsConfigured() bool {
	return len(c.Certificate.Public) > 0 && len(c.Certificate.Private) > 0
}

// Redacted implements the redactor interface used by the tee publisher
func (c Config) Redacted() any {
	return Config{
		URI_:       c.URI_,
		Port:       c.Port,
		ShipID:     c.ShipID,
		Interfaces: c.Interfaces,
		Certificate: Certificate{
			Public:  c.Certificate.Public,
			Private: util.Masked(c.Certificate.Private),
		},
	}
}

func createShipID() string {
	protectedID := machine.ProtectedID("evcc-eebus")
	return fmt.Sprintf("%s-%0x", "EVCC", protectedID[:8])
}

func DefaultConfig(conf *Config) (*Config, error) {
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

// Ski returns the EEbus server SKI
func Ski() string {
	if Instance == nil {
		return ""
	}
	return Instance.ski
}
