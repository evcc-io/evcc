package configure

import (
	"fmt"

	"github.com/evcc-io/evcc/charger/eebus"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/util"
)

// configureEEBus setup EEBus
func (c *CmdConfigure) configureEEBus(other map[string]interface{}) error {
	conf := eebus.Config{
		URI: ":4712",
	}

	if err := util.DecodeOther(other, &conf); err != nil {
		return err
	}

	srv, err := eebus.NewServer(conf)
	if err == nil {
		eebus.Instance = srv
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
		return eebusConfig, fmt.Errorf("%s", c.localizedString("Error_EEBUS_Certificate_Create"))
	}

	pubKey, privKey, err := eebus.GetX509KeyPair(cert)
	if err != nil {
		return eebusConfig, fmt.Errorf("%s", c.localizedString("Error_EEBUS_Certificate_Use"))
	}

	eebusConfig = map[string]interface{}{
		"certificate": map[string]string{
			"public":  pubKey,
			"private": privKey,
		},
	}

	return eebusConfig, nil
}
