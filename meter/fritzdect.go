package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/fritz"
	"github.com/evcc-io/evcc/meter/fritz/aha"
	"github.com/evcc-io/evcc/meter/fritz/smarthome"
	"github.com/evcc-io/evcc/util"
)

// AVM FritzBox AHA interface specifications:
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf
// https://fritz.support/resources/SmarthomeRestApiFRITZOS82.html (REST API for FritzOS 8.2+)

func init() {
	registry.Add("fritzdect", NewFritzDECTFromConfig)
}

// NewFritzDECTFromConfig creates a fritzdect meter from generic config
func NewFritzDECTFromConfig(other map[string]any) (api.Meter, error) {
	var cc fritz.Settings
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	// Use new REST API if firmware82 is set, otherwise use legacy LUA API
	if cc.Firmware82 {
		return smarthome.NewConnection(cc.URI, cc.AIN, cc.User, cc.Password)
	}

	return aha.NewConnection(cc.URI, cc.AIN, cc.User, cc.Password)
}
