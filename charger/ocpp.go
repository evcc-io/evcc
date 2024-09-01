package charger

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// OCPP charger implementation
type OCPP struct {
	log     *util.Logger
	cp      *ocpp.CP
	conn    *ocpp.Connector
	idtag   string
	phases  int
	enabled bool
	current float64

	timeout        time.Duration
	remoteStart    bool
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
		Timeout        time.Duration // Message Timeout

		BootNotification *bool                      // TODO deprecated
		GetConfiguration *bool                      // TODO deprecated
		ChargingRateUnit types.ChargingRateUnitType // TODO deprecated
		AutoStart        bool                       // TODO deprecated
		NoStop           bool                       // TODO deprecated

		StackLevelZero *bool
		RemoteStart    bool
	}{
		Connector:      1,
		IdTag:          defaultIdTag,
		MeterInterval:  10 * time.Second,
		ConnectTimeout: ocppConnectTimeout,
		Timeout:        ocppTimeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	stackLevelZero := cc.StackLevelZero != nil && *cc.StackLevelZero

	c, err := NewOCPP(cc.StationId, cc.Connector, cc.IdTag,
		cc.MeterValues, cc.MeterInterval,
		stackLevelZero, cc.RemoteStart,
		cc.ConnectTimeout, cc.Timeout)
	if err != nil {
		return c, err
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

	// var currentG func() (float64, error)
	// if c.cp.HasMeasurement(types.MeasurandCurrentOffered) {
	// 	currentG = c.conn.GetMaxCurrent
	// }

	return decorateOCPP(c, powerG, totalEnergyG, currentsG, voltagesG, phasesS, socG), nil
}

//go:generate go run ../cmd/tools/decorate.go -f decorateOCPP -b *OCPP -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Battery,Soc,func() (float64, error)"

// NewOCPP creates OCPP charger
func NewOCPP(id string, connector int, idtag string,
	meterValues string, meterInterval time.Duration,
	stackLevelZero, remoteStart bool,
	connectTimeout, timeout time.Duration,
) (*OCPP, error) {
	unit := "ocpp"
	if id != "" {
		unit = id
	}
	unit = fmt.Sprintf("%s-%d", unit, connector)

	log := util.NewLogger(unit)

	cp, err := ocpp.Instance().ChargepointByID(id)
	if err != nil {
		cp = ocpp.NewChargePoint(log, id)

		// should not error
		if err := ocpp.Instance().Register(id, cp); err != nil {
			return nil, err
		}

		log.DEBUG.Printf("waiting for chargepoint: %v", connectTimeout)

		select {
		case <-time.After(connectTimeout):
			return nil, api.ErrTimeout
		case <-cp.HasConnected():
		}

		if err := cp.Setup(meterValues, meterInterval, timeout); err != nil {
			return nil, err
		}
	}

	if cp.NumberOfConnectors > 0 && connector > cp.NumberOfConnectors {
		return nil, fmt.Errorf("invalid connector: %d", connector)
	}

	conn, err := ocpp.NewConnector(log, connector, cp, timeout)
	if err != nil {
		return nil, err
	}

	if idtag == defaultIdTag && cp.IdTag != "" {
		idtag = cp.IdTag
	}

	c := &OCPP{
		log:            log,
		cp:             cp,
		conn:           conn,
		idtag:          idtag,
		remoteStart:    remoteStart,
		stackLevelZero: stackLevelZero,
		timeout:        timeout,
	}

	if cp.HasRemoteTriggerFeature {
		if err := conn.TriggerMessageRequest(core.StatusNotificationFeatureName); err != nil {
			c.log.DEBUG.Printf("failed triggering StatusNotification: %v", err)
		}

		go conn.WatchDog(10 * time.Second)
	}

	return c, conn.Initialized()
}

// Connector returns the connector instance
func (c *OCPP) Connector() *ocpp.Connector {
	return c.conn
}

func (c *OCPP) effectiveIdTag() string {
	if idtag := c.conn.IdTag(); idtag != "" {
		return idtag
	}
	return c.idtag
}

// wait waits for a CP roundtrip with timeout
func (c *OCPP) wait(err error, rc chan error) error {
	return ocpp.Wait(err, rc, c.timeout)
}

// Status implements the api.Charger interface
func (c *OCPP) Status() (api.ChargeStatus, error) {
	status, err := c.conn.Status()
	if err != nil {
		return api.StatusNone, err
	}

	if c.conn.NeedsAuthentication() {
		if c.remoteStart {
			// lock the cable by starting remote transaction after vehicle connected
			if err := c.initTransaction(); err != nil {
				c.log.WARN.Printf("failed to start remote transaction: %v", err)
			}
		} else {
			// TODO: bring this status to UI
			c.log.WARN.Printf("waiting for local authentication")
		}
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
	if v, err := c.getScheduleLimit(); err == nil {
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

func (c *OCPP) initTransaction() error {
	rc := make(chan error, 1)
	err := ocpp.Instance().RemoteStartTransaction(c.cp.ID(), func(resp *core.RemoteStartTransactionConfirmation, err error) {
		if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
			err = errors.New(string(resp.Status))
		}

		rc <- err
	}, c.effectiveIdTag(), func(request *core.RemoteStartTransactionRequest) {
		connector := c.conn.ID()
		request.ConnectorId = &connector
	})

	return c.wait(err, rc)
}

func (c *OCPP) setChargingProfile(profile *types.ChargingProfile) error {
	rc := make(chan error, 1)
	err := ocpp.Instance().SetChargingProfile(c.cp.ID(), func(resp *smartcharging.SetChargingProfileConfirmation, err error) {
		if err == nil && resp != nil && resp.Status != smartcharging.ChargingProfileStatusAccepted {
			err = errors.New(string(resp.Status))
		}

		rc <- err
	}, c.conn.ID(), profile)

	return c.wait(err, rc)
}

// setCurrent sets the TxDefaultChargingProfile with given current
func (c *OCPP) setCurrent(current float64) error {
	err := c.setChargingProfile(c.createTxDefaultChargingProfile(math.Trunc(10*current) / 10))
	if err != nil {
		err = fmt.Errorf("set charging profile: %w", err)
	}

	return err
}

// getScheduleLimit queries the current or power limit the charge point is currently set to offer
func (c *OCPP) getScheduleLimit() (float64, error) {
	const duration = 60 // duration of requested schedule in seconds

	var limit float64

	rc := make(chan error, 1)
	err := ocpp.Instance().GetCompositeSchedule(c.cp.ID(), func(resp *smartcharging.GetCompositeScheduleConfirmation, err error) {
		if err == nil && resp != nil && resp.Status != smartcharging.GetCompositeScheduleStatusAccepted {
			err = errors.New(string(resp.Status))
		}

		if err == nil {
			if resp.ChargingSchedule != nil && len(resp.ChargingSchedule.ChargingSchedulePeriod) > 0 {
				// return first (current) period limit
				limit = resp.ChargingSchedule.ChargingSchedulePeriod[0].Limit
			} else {
				err = fmt.Errorf("invalid ChargingSchedule")
			}
		}

		rc <- err
	}, c.conn.ID(), duration)

	err = c.wait(err, rc)

	return limit, err
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
			StartSchedule:          types.Now(),
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
	rc := make(chan error, 1)
	err := ocpp.Instance().GetConfiguration(c.cp.ID(), func(resp *core.GetConfigurationConfirmation, err error) {
		if err == nil {
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

		rc <- err
	}, nil)
	c.wait(err, rc)
}

var _ loadpoint.Controller = (*OCPP)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *OCPP) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}
