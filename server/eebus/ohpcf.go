package eebus

import (
	"errors"
	"strconv"

	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/usecases/cem/ohpcf"
	spineapi "github.com/enbility/spine-go/api"
)

// onOHPCFEvent reads the heat pump compressor flexibility (OHPCF) data announced
// by a remote compressor and forwards the event to any registered devices.
// evcc currently only surfaces this data read-only (no scheduling/control).
func (c *EEBus) onOHPCFEvent(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	switch event {
	case ohpcf.DataUpdateRequestedPowerEstimate,
		ohpcf.DataUpdateRequestedPowerMax,
		ohpcf.DataUpdateConsumptionIsStoppable,
		ohpcf.DataUpdateConsumptionIsPausable,
		ohpcf.DataUpdateConsumptionStartTime,
		ohpcf.DataUpdateConsumptionState,
		ohpcf.DataUpdateMinimalRunDuration,
		ohpcf.DataUpdateMinimalPauseDuration:
		c.logOHPCF(ski, entity)
	}

	c.ucCallback(ski, device, entity, event)
}

// logOHPCF reads and logs the announced optional power consumption of a remote
// heat pump compressor
func (c *EEBus) logOHPCF(ski string, entity spineapi.EntityRemoteInterface) {
	uc := c.cem.OHPCF

	info, err := uc.OptionalPowerConsumption(entity)
	if err != nil {
		if !errors.Is(err, eebusapi.ErrDataNotAvailable) {
			c.log.DEBUG.Printf("ohpcf %s: %v", ski, err)
		}
		return
	}

	available, _ := uc.OptionalPowerConsumptionAvailable(entity)
	state, _ := uc.PowerConsumptionProcessState(entity)
	runMin, _ := uc.PowerConsumptionMinimalRunDuration(entity)
	pauseMin, _ := uc.PowerConsumptionMinimalPauseDuration(entity)

	c.log.DEBUG.Printf("ohpcf %s: available=%t state=%s power=%s maxPower=%s pausable=%t stoppable=%t minRun=%s minPause=%s",
		ski, available, state,
		fmtPower(info.Power), fmtPower(info.MaxPower),
		info.IsPausable, info.IsStoppable, runMin, pauseMin)
}

// fmtPower formats an optional power value for logging
func fmtPower(p *float64) string {
	if p == nil {
		return "n/a"
	}
	return strconv.FormatFloat(*p, 'f', -1, 64) + "W"
}
