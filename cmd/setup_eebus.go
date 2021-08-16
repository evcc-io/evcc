// +build eebus

package cmd

import (
	"github.com/andig/evcc/server"
)

// setup EEBus
func configureEEBus(conf map[string]interface{}) error {
	var err error
	if server.EEBusInstance, err = server.NewEEBus(conf); err == nil {
		go server.EEBusInstance.Run()
	}

	return nil
}
