package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/homematic"
	"github.com/evcc-io/evcc/util"
)

// AVM FritzBox AHA interface specifications:
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf

func init() {
	registry.Add("homematic", NewCCUFromConfig)
}

// NewCCUFromConfig creates a homematic meter from generic config
func NewCCUFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := &homematic.Settings{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return homematic.NewConnection(cc.URI, cc.Device, cc.MeterChannel, cc.SwitchChannel, cc.User, cc.Password)
}
