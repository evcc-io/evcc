package charger

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/charger/ocpp20"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
	"github.com/samber/lo"
)

// OCPP20 charger implementation for OCPP 2.0.1
type OCPP20 struct {
	log     *util.Logger
	cp      *ocpp20.Station
	evse    *ocpp20.EVSE
	phases  int
	enabled bool
	current float64

	stackLevelZero      bool
	profileKindRelative bool
	lp                  loadpoint.API
}

const defaultIdTag20 = "evcc" // RemoteStartTransaction only

func init() {
	registry.AddCtx("ocpp20", NewOCPP20FromConfig)
}

// NewOCPP20FromConfig creates an OCPP 2.0.1 charger from generic config
func NewOCPP20FromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		StationId      string
		IdTag          string
		EVSE           int
		Connector      int
		MeterInterval  time.Duration
		ConnectTimeout time.Duration // Initial Timeout

		StackLevelZero      *bool
		ProfileKindRelative bool
		RemoteStart         bool
		PhaseSwitching      bool
	}{
		EVSE:           1,
		Connector:      1,
		MeterInterval:  10 * time.Second,
		ConnectTimeout: 5 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	stackLevelZero := cc.StackLevelZero != nil && *cc.StackLevelZero
	profileKindRelative := cc.ProfileKindRelative

	c, err := NewOCPP20(ctx,
		cc.StationId, cc.EVSE, cc.Connector, cc.IdTag,
		cc.MeterInterval,
		stackLevelZero, profileKindRelative, cc.RemoteStart,
		cc.PhaseSwitching,
		cc.ConnectTimeout)
	if err != nil {
		return c, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	var (
		powerG, totalEnergyG, socG func() (float64, error)
		currentsG, voltagesG       func() (float64, float64, float64, error)
	)

	// Power measurement
	if _, err := c.evse.CurrentPower(); err == nil {
		powerG = c.evse.CurrentPower
	}

	// Energy measurement
	if _, err := c.evse.TotalEnergy(); err == nil {
		totalEnergyG = c.evse.TotalEnergy
	}

	// Current measurement
	if i1, i2, i3, err := c.evse.Currents(); err == nil && (i1 != 0 || i2 != 0 || i3 != 0) {
		currentsG = c.evse.Currents
	}

	// Voltage measurement
	if v1, v2, v3, err := c.evse.Voltages(); err == nil && (v1 != 0 || v2 != 0 || v3 != 0) {
		voltagesG = c.evse.Voltages
	}

	// SOC measurement
	if _, err := c.evse.Soc(); err == nil {
		socG = c.evse.Soc
	}

	// Current getter
	var currentG func() (float64, error)
	if _, err := c.evse.GetMaxCurrent(); err == nil {
		currentG = c.evse.GetMaxCurrent
	}

	// Phase switching is wired when either GetVariables discovery or manual config enables it.
	var phasesS func(int) error
	if c.cp.PhaseSwitching {
		phasesS = c.phases1p3p
	}

	return decorateOCPP20(c, powerG, totalEnergyG, currentsG, voltagesG, currentG, phasesS, socG), nil
}

//go:generate go tool decorate -f decorateOCPP20 -b *OCPP20 -r api.Charger -t api.Meter,api.MeterEnergy,api.PhaseCurrents,api.PhaseVoltages,api.CurrentGetter,api.PhaseSwitcher,api.Battery

// NewOCPP20 creates an OCPP 2.0.1 charger
func NewOCPP20(ctx context.Context,
	id string, evseID, connectorID int, idTag string,
	meterInterval time.Duration,
	stackLevelZero, profileKindRelative, remoteStart, phaseSwitching bool,
	connectTimeout time.Duration,
) (*OCPP20, error) {
	log := util.NewLogger(fmt.Sprintf("%s-%d", lo.CoalesceOrEmpty(id, "ocpp20"), evseID))

	cp, err := ocpp20.Instance().RegisterChargingStation(id,
		func() *ocpp20.Station {
			return ocpp20.NewStation(log, id)
		},
		func(cp *ocpp20.Station) error {
			log.DEBUG.Printf("waiting for charging station: %v", connectTimeout)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(connectTimeout):
				return api.ErrTimeout
			case <-cp.HasConnected():
			}

			return cp.Setup()
		},
	)
	if err != nil {
		return nil, err
	}

	if remoteStart {
		idTag = lo.CoalesceOrEmpty(idTag, defaultIdTag20)
	}

	evse, err := ocpp20.NewEVSE(ctx, log, evseID, connectorID, cp, idTag, meterInterval)
	if err != nil {
		return nil, err
	}

	// manual config force-enables; never disables a discovered capability
	if phaseSwitching {
		cp.EnablePhaseSwitching()
	}

	c := &OCPP20{
		log:                 log,
		cp:                  cp,
		evse:                evse,
		stackLevelZero:      stackLevelZero,
		profileKindRelative: profileKindRelative,
	}

	return c, evse.Initialized()
}

// phases1p3p implements the api.PhaseSwitcher interface.
// In OCPP 2.0.1, phase switching is encoded via ChargingSchedulePeriod.NumberPhases
// in the active charging profile, so we re-send the profile with the new phase count.
func (c *OCPP20) phases1p3p(phases int) error {
	c.phases = phases

	enabled, err := c.Enabled()
	if err != nil {
		return err
	}

	var current float64
	if enabled {
		current = c.current
	}

	return c.setCurrent(current)
}

// EVSE returns the EVSE instance
func (c *OCPP20) EVSE() *ocpp20.EVSE {
	return c.evse
}

// Status implements the api.Charger interface
func (c *OCPP20) Status() (api.ChargeStatus, error) {
	status, err := c.evse.Status()
	if err != nil {
		return api.StatusNone, err
	}

	// Get charging state from transaction
	chargingState, hasTransaction := c.evse.ChargingState()

	switch status {
	case availability.ConnectorStatusAvailable,
		availability.ConnectorStatusUnavailable,
		availability.ConnectorStatusReserved:
		return api.StatusA, nil

	case availability.ConnectorStatusOccupied:
		// Check charging state
		if hasTransaction {
			switch chargingState {
			case transactions.ChargingStateCharging:
				return api.StatusC, nil
			case transactions.ChargingStateEVConnected,
				transactions.ChargingStateSuspendedEV,
				transactions.ChargingStateSuspendedEVSE,
				transactions.ChargingStateIdle:
				return api.StatusB, nil
			}
		}
		return api.StatusB, nil

	case availability.ConnectorStatusFaulted:
		return api.StatusNone, fmt.Errorf("charger fault")

	default:
		return api.StatusNone, fmt.Errorf("invalid status: %s", status)
	}
}

var _ api.StatusReasoner = (*OCPP20)(nil)

func (c *OCPP20) StatusReason() (api.Reason, error) {
	var res api.Reason

	if c.evse.NeedsAuthentication() {
		res = api.ReasonWaitingForAuthorization
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (c *OCPP20) Enabled() (bool, error) {
	if s, err := c.evse.Status(); err == nil {
		switch s {
		case availability.ConnectorStatusUnavailable:
			return false, nil
		case availability.ConnectorStatusOccupied:
			// Check if actually charging
			if state, ok := c.evse.ChargingState(); ok {
				return state == transactions.ChargingStateCharging, nil
			}
			return true, nil
		}
	}

	// fallback to the "offered" measurands
	if v, err := c.evse.GetMaxCurrent(); err == nil {
		return v > 0, nil
	}
	if v, err := c.evse.GetMaxPower(); err == nil {
		return v > 0, nil
	}

	// fallback to cached value as last resort
	return c.enabled, nil
}

// Enable implements the api.Charger interface
func (c *OCPP20) Enable(enable bool) error {
	var current float64
	if enable {
		current = c.current
	}

	err := c.setCurrent(current)
	if err == nil {
		// cache enabled state as last fallback option
		c.enabled = enable
	}

	return err
}

// setCurrent sets the TxDefaultChargingProfile with given current
func (c *OCPP20) setCurrent(current float64) error {
	err := c.setChargingProfile(c.createTxDefaultChargingProfile(math.Trunc(10*current) / 10))
	if err != nil {
		err = fmt.Errorf("set charging profile: %w", err)
	}

	return err
}

// setChargingProfile sends a SetChargingProfile request
func (c *OCPP20) setChargingProfile(profile *types.ChargingProfile) error {
	rc := make(chan error, 1)

	err := ocpp20.Instance().SetChargingProfile(
		c.cp.ID(),
		func(resp *smartcharging.SetChargingProfileResponse, err error) {
			if err == nil && resp.Status != smartcharging.ChargingProfileStatusAccepted {
				err = fmt.Errorf("status: %s", resp.Status)
			}
			rc <- err
		},
		c.evse.ID(),
		profile,
	)

	if err != nil {
		return err
	}

	select {
	case err := <-rc:
		return err
	case <-time.After(ocpp.Timeout):
		return api.ErrTimeout
	}
}

// createTxDefaultChargingProfile returns a TxDefaultChargingProfile with given current
func (c *OCPP20) createTxDefaultChargingProfile(current float64) *types.ChargingProfile {
	phases := c.phases
	period := types.ChargingSchedulePeriod{
		StartPeriod: 0,
		Limit:       current,
	}

	if c.cp.ChargingRateUnit == types.ChargingRateUnitWatts {
		period.Limit = math.Trunc(230.0 * current * float64(phases))
	} else {
		// OCPP assumes phases == 3 if not set
		if phases != 0 {
			// set explicit phase configuration
			period.NumberPhases = &phases
		}
	}

	scheduleID := 1

	res := &types.ChargingProfile{
		ID:                     c.cp.ChargingProfileId,
		StackLevel:             c.cp.StackLevel,
		ChargingProfilePurpose: types.ChargingProfilePurposeTxDefaultProfile,
		ChargingProfileKind:    types.ChargingProfileKindAbsolute,
		ChargingSchedule: []types.ChargingSchedule{{
			ID:                     scheduleID,
			ChargingRateUnit:       c.cp.ChargingRateUnit,
			ChargingSchedulePeriod: []types.ChargingSchedulePeriod{period},
		}},
	}

	if c.profileKindRelative {
		res.ChargingProfileKind = types.ChargingProfileKindRelative
	} else {
		startSchedule := types.NewDateTime(time.Now().Add(-time.Minute))
		res.ChargingSchedule[0].StartSchedule = startSchedule
	}

	if c.stackLevelZero {
		res.StackLevel = 0
	}

	return res
}

// MaxCurrent implements the api.Charger interface
func (c *OCPP20) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*OCPP20)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *OCPP20) MaxCurrentMillis(current float64) error {
	err := c.setCurrent(current)
	if err == nil {
		c.current = current
	}
	return err
}

var _ api.Identifier = (*OCPP20)(nil)

// Identify implements the api.Identifier interface
func (c *OCPP20) Identify() (string, error) {
	return c.evse.IdTag(), nil
}

var _ api.Diagnosis = (*OCPP20)(nil)

// Diagnose implements the api.Diagnosis interface
func (c *OCPP20) Diagnose() {
	fmt.Printf("\tCharging Station ID: %s\n", c.cp.ID())

	if c.cp.BootNotificationResult != nil {
		fmt.Printf("\tBoot Notification:\n")
		fmt.Printf("\t\tVendor: %s\n", c.cp.BootNotificationResult.ChargingStation.VendorName)
		fmt.Printf("\t\tModel: %s\n", c.cp.BootNotificationResult.ChargingStation.Model)
		if c.cp.BootNotificationResult.ChargingStation.SerialNumber != "" {
			fmt.Printf("\t\tSerialNumber: %s\n", c.cp.BootNotificationResult.ChargingStation.SerialNumber)
		}
		if c.cp.BootNotificationResult.ChargingStation.FirmwareVersion != "" {
			fmt.Printf("\t\tFirmwareVersion: %s\n", c.cp.BootNotificationResult.ChargingStation.FirmwareVersion)
		}
	}

	// TODO: Add GetVariables-based configuration dump
	fmt.Printf("\tConfiguration: (GetVariables not yet implemented)\n")
}

var _ loadpoint.Controller = (*OCPP20)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *OCPP20) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}
