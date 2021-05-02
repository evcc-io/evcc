// +build eebus

package eebus

import (
	"os"
	"sync"

	"github.com/amp-x/eebus/communication"
	"github.com/andig/evcc/util"
)

// Instance is the EEBUS CEM listener instance
// This is needed since EEBUS CEM is managing all the chargers its connected to
var Instance *Cem

// Cem singleton implementing the EEBUS CEM
type Cem struct {
	mux     sync.Mutex
	log     *util.Logger
	key     string
	cert    string
	eebusSC *communication.ServiceController
}

// New creates the CEM
func New(log *util.Logger, key, cert string) (*Cem, error) {
	l := &Cem{
		log:  log,
		key:  key,
		cert: cert,
	}

	deviceDetails := communication.ManufacturerDetailsType{
		DeviceName:    "EVCC",
		DeviceCode:    "EVCC_HEMS_01",
		DeviceAddress: "EVCC_HEMS",
		BrandName:     "EVCC",
	}

	certData := &communication.CertificateBase64Encoded{
		Public:  cert,
		Private: key,
	}
	l.eebusSC = communication.NewServiceController(log.TRACE, deviceDetails, certData)

	go func() {
		if err := l.eebusSC.Boot(); err != nil {
			log.FATAL.Fatal(err)
			os.Exit(1)
		}
	}()
	return l, nil
}

// EEBUS ServiceController send new loadControl currents
func (c *Cem) SetCurrents(ski string, currentL1, currentL2, currentL3 float64) error {
	return c.eebusSC.SetCurrents(ski, []float64{currentL1, currentL2, currentL3})
}

// EEBUS ServiceController get current dataset
func (c *Cem) GetData(ski string) (*communication.EVSEClientDataType, error) {
	return c.eebusSC.GetEVSEClientData(ski)
}
