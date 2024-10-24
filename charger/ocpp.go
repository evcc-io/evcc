package charger

// LICENSE

// Copyright (c) 2024 premultiply, andig

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
	"cmp"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/samber/lo"
)

// OCPP charger implementation
type OCPP struct {
	log     *util.Logger
	cp      *ocpp.CP
	conn    *ocpp.Connector
	phases  int
	enabled bool
	current float64

	stackLevelZero bool
	lp             loadpoint.API
}

const defaultIdTag = "evcc" // RemoteStartTransaction only

func init() {
	registry.Add("ocpp", NewOCPPFromConfig)
}

// NewOCPPFromConfig creates a OCPP charger from generic config
func NewOCPPFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		StationId      string
		IdTag          string
		Connector      int
		MeterInterval  time.Duration
		MeterValues    string
		ConnectTimeout time.Duration // Initial Timeout

		Timeout          time.Duration              // TODO deprecated
		BootNotification *bool                      // TODO deprecated
		GetConfiguration *bool                      // TODO deprecated
		ChargingRateUnit types.ChargingRateUnitType // TODO deprecated
		AutoStart        bool                       // TODO deprecated
		NoStop           bool                       // TODO deprecated

		StackLevelZero *bool
		RemoteStart    bool
	}{
		Connector:      1,
		MeterInterval:  10 * time.Second,
		ConnectTimeout: 5 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	stackLevelZero := cc.StackLevelZero != nil && *cc.StackLevelZero

	c, err := NewOCPP(cc.StationId, cc.Connector, cc.IdTag,
		cc.MeterValues, cc.MeterInterval,
		stackLevelZero, cc.RemoteStart,
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

	if c.cp.HasMeasurement(types.MeasurandPowerActiveImport) {
		powerG = c.conn.CurrentPower
	}

	if c.cp.HasMeasurement(types.MeasurandEnergyActiveImportRegister) {
		totalEnergyG = c.conn.TotalEnergy
	}

	if c.cp.HasMeasurement(types.MeasurandCurrentImport) {
		currentsG = c.conn.Currents
	}

	if c.cp.HasMeasurement(types.MeasurandVoltage) {
		voltagesG = c.conn.Voltages
	}

	if c.cp.HasMeasurement(types.MeasurandSoC) {
		socG = c.conn.Soc
	}

	var phasesS func(int) error
	if c.cp.PhaseSwitching {
		phasesS = c.phases1p3p
	}

	var currentG func() (float64, error)
	if c.cp.HasMeasurement(types.MeasurandCurrentOffered) {
		currentG = c.conn.GetMaxCurrent
	}

	return decorateOCPP(c, powerG, totalEnergyG, currentsG, voltagesG, currentG, phasesS, socG), nil
}

//go:generate go run ../cmd/tools/decorate.go -f decorateOCPP -b *OCPP -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.CurrentGetter,GetMaxCurrent,func() (float64, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Battery,Soc,func() (float64, error)"

// NewOCPP creates OCPP charger
func NewOCPP(id string, connector int, idTag string,
	meterValues string, meterInterval time.Duration,
	stackLevelZero, remoteStart bool,
	connectTimeout time.Duration,
) (*OCPP, error) {
	log := util.NewLogger(fmt.Sprintf("%s-%d", lo.CoalesceOrEmpty(id, "ocpp"), connector))

	cp, err := ocpp.Instance().RegisterChargepoint(id,
		func() *ocpp.CP {
			return ocpp.NewChargePoint(log, id)
		},
		func(cp *ocpp.CP) error {
			log.DEBUG.Printf("waiting for chargepoint: %v", connectTimeout)

			select {
			case <-time.After(connectTimeout):
				return api.ErrTimeout
			case <-cp.HasConnected():
			}

			return cp.Setup(meterValues, meterInterval)
		},
	)
	if err != nil {
		return nil, err
	}

	if cp.NumberOfConnectors > 0 && connector > cp.NumberOfConnectors {
		return nil, fmt.Errorf("invalid connector: %d", connector)
	}

	if remoteStart {
		idTag = lo.CoalesceOrEmpty(idTag, cp.IdTag, defaultIdTag)
	}

	conn, err := ocpp.NewConnector(log, connector, cp, idTag)
	if err != nil {
		return nil, err
	}

	c := &OCPP{
		log:            log,
		cp:             cp,
		conn:           conn,
		stackLevelZero: stackLevelZero,
	}

	go conn.WatchDog(10 * time.Second)

	return c, conn.Initialized()
}

// Connector returns the connector instance
func (c *OCPP) Connector() *ocpp.Connector {
	return c.conn
}

// Status implements the api.Charger interface
func (c *OCPP) Status() (api.ChargeStatus, error) {
	status, err := c.conn.Status()
	if err != nil {
		return api.StatusNone, err
	}

	switch status {
	case
		core.ChargePointStatusAvailable,   // "Available"
		core.ChargePointStatusUnavailable: // "Unavailable"
		return api.StatusA, nil
	case
		core.ChargePointStatusPreparing,     // "Preparing"
		core.ChargePointStatusSuspendedEVSE, // "SuspendedEVSE"
		core.ChargePointStatusSuspendedEV,   // "SuspendedEV"
		core.ChargePointStatusFinishing:     // "Finishing"
		return api.StatusB, nil
	case
		core.ChargePointStatusCharging: // "Charging"
		return api.StatusC, nil
	case
		core.ChargePointStatusReserved, // "Reserved"
		core.ChargePointStatusFaulted:  // "Faulted"
		return api.StatusF, fmt.Errorf("chargepoint status: %s", status)
	default:
		return api.StatusNone, fmt.Errorf("invalid chargepoint status: %s", status)
	}
}

var _ api.StatusReasoner = (*OCPP)(nil)

func (c *OCPP) StatusReason() (api.Reason, error) {
	var res api.Reason

	s, err := c.conn.Status()
	if err != nil {
		return res, err
	}

	switch {
	case c.conn.NeedsAuthentication():
		res = api.ReasonWaitingForAuthorization

	case s == core.ChargePointStatusFinishing:
		res = api.ReasonDisconnectRequired
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (c *OCPP) Enabled() (bool, error) {
	if s, err := c.conn.Status(); err == nil {
		switch s {
		case
			core.ChargePointStatusSuspendedEVSE:
			return false, nil
		case
			core.ChargePointStatusCharging,
			core.ChargePointStatusSuspendedEV:
			return true, nil
		}
	}

	// fallback to the "offered" measurands
	if c.cp.HasMeasurement(types.MeasurandCurrentOffered) {
		if v, err := c.conn.GetMaxCurrent(); err == nil {
			return v > 0, nil
		}
	}
	if c.cp.HasMeasurement(types.MeasurandPowerOffered) {
		if v, err := c.conn.GetMaxPower(); err == nil {
			return v > 0, nil
		}
	}

	// fallback to querying the active charging profile schedule limit
	if v, err := c.conn.GetScheduleLimit(60); err == nil {
		return v > 0, nil
	}

	// fallback to cached value as last resort
	return c.enabled, nil
}

// Enable implements the api.Charger interface
func (c *OCPP) Enable(enable bool) error {
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
func (c *OCPP) setCurrent(current float64) error {
	err := c.conn.SetChargingProfileRequest(c.createTxDefaultChargingProfile(math.Trunc(10*current) / 10))
	if err != nil {
		err = fmt.Errorf("set charging profile: %w", err)
	}

	return err
}

// createTxDefaultChargingProfile returns a TxDefaultChargingProfile with given current
func (c *OCPP) createTxDefaultChargingProfile(current float64) *types.ChargingProfile {
	phases := c.phases
	period := types.NewChargingSchedulePeriod(0, current)
	if c.cp.ChargingRateUnit == types.ChargingRateUnitWatts {
		// get (expectedly) active phases from loadpoint
		if c.lp != nil {
			phases = c.lp.GetPhases()
		}
		if phases == 0 {
			phases = 3
		}
		period = types.NewChargingSchedulePeriod(0, math.Trunc(230.0*current*float64(phases)))
	}

	// OCPP assumes phases == 3 if not set
	if phases != 0 {
		period.NumberPhases = &phases
	}

	res := &types.ChargingProfile{
		ChargingProfileId:      c.cp.ChargingProfileId,
		ChargingProfilePurpose: types.ChargingProfilePurposeTxDefaultProfile,
		ChargingProfileKind:    types.ChargingProfileKindAbsolute,
		ChargingSchedule: &types.ChargingSchedule{
			StartSchedule:          types.NewDateTime(time.Now().Add(-time.Minute)),
			ChargingRateUnit:       c.cp.ChargingRateUnit,
			ChargingSchedulePeriod: []types.ChargingSchedulePeriod{period},
		},
	}

	if !c.stackLevelZero {
		res.StackLevel = c.cp.StackLevel
	}

	return res
}

// MaxCurrent implements the api.Charger interface
func (c *OCPP) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*OCPP)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *OCPP) MaxCurrentMillis(current float64) error {
	err := c.setCurrent(current)
	if err == nil {
		c.current = current
	}
	return err
}

// phases1p3p implements the api.PhaseSwitcher interface
func (c *OCPP) phases1p3p(phases int) error {
	c.phases = phases

	return c.setCurrent(c.current)
}

var _ api.Identifier = (*OCPP)(nil)

// Identify implements the api.Identifier interface
func (c *OCPP) Identify() (string, error) {
	return c.conn.IdTag(), nil
}

var _ api.Diagnosis = (*OCPP)(nil)

// Diagnose implements the api.Diagnosis interface
func (c *OCPP) Diagnose() {
	fmt.Printf("\tCharge Point ID: %s\n", c.cp.ID())

	if c.cp.BootNotificationResult != nil {
		fmt.Printf("\tBoot Notification:\n")
		fmt.Printf("\t\tChargePointVendor: %s\n", c.cp.BootNotificationResult.ChargePointVendor)
		fmt.Printf("\t\tChargePointModel: %s\n", c.cp.BootNotificationResult.ChargePointModel)
		fmt.Printf("\t\tChargePointSerialNumber: %s\n", c.cp.BootNotificationResult.ChargePointSerialNumber)
		fmt.Printf("\t\tFirmwareVersion: %s\n", c.cp.BootNotificationResult.FirmwareVersion)
	}

	fmt.Printf("\tConfiguration:\n")
	if resp, err := c.cp.GetConfigurationRequest(); err == nil {
		// sort configuration keys for printing
		slices.SortFunc(resp.ConfigurationKey, func(i, j core.ConfigurationKey) int {
			return cmp.Compare(i.Key, j.Key)
		})

		rw := map[bool]string{false: "r/w", true: "r/o"}

		for _, opt := range resp.ConfigurationKey {
			if opt.Value == nil {
				continue
			}

			fmt.Printf("\t\t%s (%s): %s\n", opt.Key, rw[opt.Readonly], *opt.Value)
		}
	}
}

var _ loadpoint.Controller = (*OCPP)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *OCPP) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}
