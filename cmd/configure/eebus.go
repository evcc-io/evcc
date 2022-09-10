package configure

import (
	"crypto/x509/pkix"
	"fmt"

	certhelper "github.com/evcc-io/eebus/cert"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/server"
)

// configureEEBus setup EEBus
func (c *CmdConfigure) configureEEBus(conf map[string]interface{}) error {
	var err error
	if server.EEBusInstance, err = server.NewEEBus(conf); err == nil {
		go server.EEBusInstance.Run()
		shutdown.Register(server.EEBusInstance.Shutdown)
	}

	return err
}

// eebusCertificate creates EEBUS certificate and returns private/public key
func (c *CmdConfigure) eebusCertificate() (map[string]interface{}, error) {
	details := server.EEBUSDetails

	subject := pkix.Name{
		CommonName:   details.DeviceCode,
		Country:      []string{"DE"},
		Organization: []string{details.BrandName},
	}

	var eebusConfig map[string]interface{}

	cert, err := certhelper.CreateCertificate(true, subject)
	if err != nil {
		return eebusConfig, fmt.Errorf("%s", c.localizedString("Error_EEBUS_Certificate_Create", nil))
	}

	pubKey, privKey, err := certhelper.GetX509KeyPair(cert)
	if err != nil {
		return eebusConfig, fmt.Errorf("%s", c.localizedString("Error_EEBUS_Certificate_Use", nil))
	}

	eebusConfig = map[string]interface{}{
		"certificate": map[string]string{
			"public":  pubKey,
			"private": privKey,
		},
	}

	return eebusConfig, nil
}
