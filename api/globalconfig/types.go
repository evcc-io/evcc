package globalconfig

import "github.com/evcc-io/evcc/provider/mqtt"

type Mqtt struct {
	mqtt.Config `mapstructure:",squash"`
	Topic       string
}
