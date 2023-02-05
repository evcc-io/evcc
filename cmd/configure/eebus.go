package configure

import (
	"fmt"

	"github.com/evcc-io/evcc/charger/eebus"
	"github.com/evcc-io/evcc/cmd/shutdown"
)

// configureEEBus setup EEBus
func (c *CmdConfigure) configureEEBus(conf map[string]interface{}) error {
	var err error
	if eebus.Instance, err = eebus.NewServer(conf); err == nil {
		go eebus.Instance.Run()
		shutdown.Register(eebus.Instance.Shutdown)
	}

	return err
}

// eebusCertificate creates EEBUS certificate and returns private/public key
func (c *CmdConfigure) eebusCertificate() (map[string]interface{}, error) {
	var eebusConfig map[string]interface{}

	cert, err := eebus.CreateCertificate()
	if err != nil {
		return eebusConfig, fmt.Errorf("%s", c.localizedString("Error_EEBUS_Certificate_Create", nil))
	}

	pubKey, privKey, err := eebus.GetX509KeyPair(cert)
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
