package charger

import (
	"errors"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

// OpenWB charger implementation
type OpenWB struct {
	log *util.Logger
	*SimpleEVSE
}

func init() {
	registry.Add("openwb", NewOpenWBFromConfig)
}

// TODO generation of setters is not yet functional
// go:generate go run ../cmd/tools/decorate.go -p charger -f decorateOpenWB -b api.Charger -o openwb_decorators -t "api.ChargePhases,Phases1p3p,func(int64) error"

// NewOpenWBFromConfig creates a OpenWB charger from generic config
func NewOpenWBFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Phases          bool `yaml:"1p3p"`
	}{
		Settings: modbus.Settings{
			Baudrate: 9600,
			Comset:   "8N1",
			ID:       1,
		},
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	evse, err := NewSimpleEVSE(cc.URI, cc.Device, cc.Settings.Comset, cc.Settings.Baudrate, true, cc.Settings.ID)
	if err != nil {
		return nil, err
	}

	return NewOpenWB(evse, cc.Phases)
}

// NewOpenWB creates OpenWB charger
func NewOpenWB(evse *SimpleEVSE, phases bool) (api.Charger, error) {
	log := util.NewLogger("openwb")
	evse.log = log

	owb := &OpenWB{
		log:        log,
		SimpleEVSE: evse,
	}

	var phasesS func(int64) error
	if phases {
		phasesS = owb.phases1p3p
	}

	return decorateOpenWB(owb, phasesS), nil
}

// Phases1p3p implements the Charger.Phases1p3p interface
func (owb *OpenWB) phases1p3p(int64) error {
	return errors.New("not implemented")
}
