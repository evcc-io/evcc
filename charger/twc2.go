package charger

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger/twc2"
	"github.com/andig/evcc/util"
)

// TWC2 is an api.ChargeController implementation for Tesla Wall Connectors Gen 2
type TWC2 struct {
	log     *util.Logger
	handler *twc2.Master
}

// NewTWC2FromConfig creates a TWC2 charger from generic config
func NewTWC2FromConfig(log *util.Logger, other map[string]interface{}) *TWC2 {
	cc := struct {
		Device string
	}{
		Device: "/dev/ttyUSB0",
	}
	util.DecodeOther(log, other, &cc)

	wb := NewTWC2(cc.Device)
	return wb
}

// NewTWC2 creates a TWC2 charger
func NewTWC2(dev string) *TWC2 {
	log := util.NewLogger("twc2")

	wb := &TWC2{
		log:     log,
		handler: twc2.NewMaster(log, dev),
	}

	wb.log.WARN.Println("-- experimental --")

	go wb.handler.Run()

	return wb
}

// Status implements the Charger.Status interface
func (wb *TWC2) Status() (api.ChargeStatus, error) {
	state := wb.handler.PlugState()
	_ = state
	return api.StatusA, nil
}

// Enabled implements the Charger.Enabled interface
func (wb *TWC2) Enabled() (bool, error) {
	return false, nil
}

// Enable implements the Charger.Enable interface
func (wb *TWC2) Enable(enable bool) error {
	return nil
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *TWC2) MaxCurrent(current int64) error {
	return nil
}
