package heating

import (
	"context"
	"errors"

	"github.com/evcc-io/evcc/api"
)

const (
	_ int64 = iota
	Normal
	Boost
	Stop
)

// SgReadyModeController controller implementation
type SgReadyModeController struct {
	_mode int64
	modeG func() (int64, error)
	modeS func(int64) error
}

// NewSgReadyModeController creates SgReady mode controller
func NewSgReadyModeController(ctx context.Context, modeS func(int64) error, modeG func() (int64, error)) *SgReadyModeController {
	return &SgReadyModeController{
		_mode: Normal,
		modeG: modeG,
		modeS: modeS,
	}
}

func (wb *SgReadyModeController) mode() (int64, error) {
	if wb.modeG == nil {
		return wb._mode, nil
	}
	return wb.modeG()
}

// Status implements the api.Charger interface
func (wb *SgReadyModeController) Status() (api.ChargeStatus, error) {
	mode, err := wb.mode()
	if err != nil {
		return api.StatusNone, err
	}

	if mode == Stop {
		return api.StatusNone, errors.New("stop mode")
	}

	status := map[int64]api.ChargeStatus{Boost: api.StatusC, Normal: api.StatusB}
	return status[mode], nil
}

// Enabled implements the api.Charger interface
func (wb *SgReadyModeController) Enabled() (bool, error) {
	mode, err := wb.mode()
	return mode == Boost, err
}

// Enable implements the api.Charger interface
func (wb *SgReadyModeController) Enable(enable bool) error {
	mode := map[bool]int64{false: Normal, true: Boost}[enable]
	err := wb.modeS(mode)
	if err == nil {
		wb._mode = mode
	}
	return err
}
