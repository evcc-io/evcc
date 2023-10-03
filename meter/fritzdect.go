package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/fritzdect"
	"github.com/evcc-io/evcc/util"
)

// AVM FritzBox AHA interface specifications:
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf

func init() {
	registry.Add("fritzdect", NewFritzDECTFromConfig)
}

// NewFritzDECTFromConfig creates a fritzdect meter from generic config
func NewFritzDECTFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc fritzdect.Settings
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return fritzdect.NewConnection(cc.URI, cc.AIN, cc.User, cc.Password)
}
