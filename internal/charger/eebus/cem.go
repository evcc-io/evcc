// +build eebus

package eebus

import (
	"crypto/tls"
	"os"

	"github.com/amp-x/eebus/communication"
	"github.com/andig/evcc/util"
)

// Instance is the EEBUS CEM listener instance
// This is needed since EEBUS CEM is managing all the chargers its connected to
var Instance *Cem

// Cem singleton implementing the EEBUS CEM
type Cem struct {
	log *util.Logger
	sc  *communication.ServiceController
}

// New creates the CEM
func New(log *util.Logger, cert tls.Certificate) (*Cem, error) {
	l := &Cem{
		log: log,
	}

	device := communication.ManufacturerDetailsType{
		DeviceName:    "EVCC",
		DeviceCode:    "EVCC_HEMS_01",
		DeviceAddress: "EVCC_HEMS",
		BrandName:     "EVCC",
	}

	l.sc = communication.NewServiceController(log.TRACE, device, cert)

	go func() {
		if err := l.sc.Boot(); err != nil {
			log.FATAL.Fatal(err)
			os.Exit(1)
		}
	}()

	return l, nil
}

// EEBUS ServiceController send new loadControl currents
func (c *Cem) SetCurrents(ski string, currentL1, currentL2, currentL3 float64) error {
	return c.sc.SetCurrents(ski, []float64{currentL1, currentL2, currentL3})
}

// EEBUS ServiceController get current dataset
func (c *Cem) GetData(ski string) (*communication.EVSEClientDataType, error) {
	return c.sc.GetEVSEClientData(ski)
}
