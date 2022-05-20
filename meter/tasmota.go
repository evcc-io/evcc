package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tasmota"
	"github.com/evcc-io/evcc/util"
)

// Tasmota meter implementation
func init() {
	registry.Add("tasmota", NewTasmotaFromConfig)
}

// NewTapoFromConfig creates a tapo meter from generic config
func NewTasmotaFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		Channel  int
	}{
		Channel: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return tasmota.NewConnection(cc.URI, cc.User, cc.Password, cc.Channel)
}
