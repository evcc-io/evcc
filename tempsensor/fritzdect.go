package tempsensor

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/meter/fritz"
	"github.com/evcc-io/evcc/meter/fritz/aha"
	"github.com/evcc-io/evcc/meter/fritz/smarthome"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("fritzdect", NewFritzDECTFromConfig)
}

// NewFritzDECTFromConfig creates a fritzdect temp sensor from generic config
func NewFritzDECTFromConfig(other map[string]any) (api.Battery, error) {
	cc := fritz.Settings{
		Unit: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.Firmware82 {
		conn, err := smarthome.NewConnection(cc.URI, cc.AIN, cc.User, cc.Password, cc.Unit)
		if err != nil {
			return nil, err
		}
		return &sensor{implement.Battery(conn.Temperature)}, nil
	}

	conn, err := aha.NewConnection(cc.URI, cc.AIN, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}
	return &sensor{implement.Battery(conn.Temperature)}, nil
}
