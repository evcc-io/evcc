package charger

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// OCPP charger implementation
type OCPP struct {
	log                     *util.Logger
	conn                    *ocpp.Connector
	idtag                   string
	phases                  int
	enabled                 bool
	current                 float64
	meterValuesSample       string
	timeout                 time.Duration
	phaseSwitching          bool
	remoteStart             bool
	hasRemoteTriggerFeature bool
	chargingRateUnit        types.ChargingRateUnitType
	chargingProfileId       int
	stackLevel              int
	lp                      loadpoint.API
	bootNotification        *core.BootNotificationRequest
}

const (
	defaultIdTag      = "evcc" // RemoteStartTransaction only
	desiredMeasurands = "Power.Active.Import,Energy.Active.Import.Register,Current.Import,Voltage,Current.Offered,Power.Offered,SoC"
)

func init() {
	registry.Add("ocpp", NewOCPPFromConfig)
}

// NewOCPPFromConfig creates a OCPP charger from generic config
func NewOCPPFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		StationId        string
		IdTag            string
		Connector        int
		MeterInterval    time.Duration
		MeterValues      string
		ConnectTimeout   time.Duration // Initial Timeout
		Timeout          time.Duration // Message Timeout
		BootNotification *bool         // TODO deprecated
		GetConfiguration *bool         // TODO deprecated
		ChargingRateUnit string        // TODO deprecated
		AutoStart        bool          // TODO deprecated
		NoStop           bool          // TODO deprecated
		RemoteStart      bool
	}{
		Connector:        1,
		IdTag:            defaultIdTag,
		MeterInterval:    10 * time.Second,
		ConnectTimeout:   ocppConnectTimeout,
		Timeout:          ocppTimeout,
		ChargingRateUnit: "A",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	boot := cc.BootNotification != nil && *cc.BootNotification
	noConfig := cc.GetConfiguration != nil && !*cc.GetConfiguration

	c, err := NewOCPP(cc.StationId, cc.Connector, cc.IdTag,
		cc.MeterValues, cc.MeterInterval,
		boot, noConfig, cc.RemoteStart,
		cc.ConnectTimeout, cc.Timeout, cc.ChargingRateUnit)
	if err != nil {
		return c, err
	}

	var (
		powerG, totalEnergyG, socG func() (float64, error)
		currentsG, voltagesG       func() (float64, float64, float64, error)
	)

	if c.hasMeasurement(types.MeasurandPowerActiveImport) {
		powerG = c.conn.CurrentPower
	}

	if c.hasMeasurement(types.MeasurandEnergyActiveImportRegister) {
		totalEnergyG = c.conn.TotalEnergy
	}

	if c.hasMeasurement(types.MeasurandCurrentImport) {
		currentsG = c.conn.Currents
	}

	if c.hasMeasurement(types.MeasurandVoltage) {
		voltagesG = c.conn.Voltages
	}

	if c.hasMeasurement(types.MeasurandSoC) {
		socG = c.conn.Soc
	}

	var phasesS func(int) error
	if c.phaseSwitching {
		phasesS = c.phases1p3p
	}

	// var currentG func() (float64, error)
	// if c.hasMeasurement(types.MeasurandCurrentOffered) {
	// 	currentG = c.conn.GetMaxCurrent
	// }

	return decorateOCPP(c, powerG, totalEnergyG, currentsG, voltagesG, phasesS, socG), nil
}

//go:generate go run ../cmd/tools/decorate.go -f decorateOCPP -b *OCPP -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Battery,Soc,func() (float64, error)"

// NewOCPP creates OCPP charger
func NewOCPP(id string, connector int, idtag string,
	meterValues string, meterInterval time.Duration,
	boot, noConfig, remoteStart bool,
	connectTimeout, timeout time.Duration,
	chargingRateUnit string,
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
	}

	conn, err := ocpp.NewConnector(log, connector, cp, timeout)
	if err != nil {
		return nil, err
	}

	c := &OCPP{
		log:         log,
		conn:        conn,
		idtag:       idtag,
		remoteStart: remoteStart,

		chargingRateUnit:        types.ChargingRateUnitType(chargingRateUnit),
		hasRemoteTriggerFeature: true, // assume remote trigger feature is available
		timeout:                 timeout,
	}

	c.log.DEBUG.Printf("waiting for chargepoint: %v", connectTimeout)

	select {
	case <-time.After(connectTimeout):
		return nil, api.ErrTimeout
	case <-cp.HasConnected():
	}

	// fix timing issue in EVBox when switching OCPP protocol version
	time.Sleep(time.Second)

	if err := ocpp.Instance().ChangeAvailabilityRequest(cp.ID(), 0, core.AvailabilityTypeOperative); err != nil {
		c.log.DEBUG.Printf("failed configuring availability: %v", err)
	}

	var meterValuesSampledData string
	meterValuesSampledDataMaxLength := len(strings.Split(desiredMeasurands, ","))

	rc := make(chan error, 1)

	// CP
	err = ocpp.Instance().GetConfiguration(cp.ID(), func(resp *core.GetConfigurationConfirmation, err error) {
		if err == nil {
			for _, opt := range resp.ConfigurationKey {
				if opt.Value == nil {
					continue
				}

				switch opt.Key {
				case ocpp.KeyChargeProfileMaxStackLevel:
					if val, err := strconv.Atoi(*opt.Value); err == nil {
						c.stackLevel = val
					}

				case ocpp.KeyChargingScheduleAllowedChargingRateUnit:
					if *opt.Value == "Power" || *opt.Value == "W" { // "W" is not allowed by spec but used by some CPs
						c.chargingRateUnit = types.ChargingRateUnitWatts
					}

				case ocpp.KeyConnectorSwitch3to1PhaseSupported:
					var val bool
					if val, err = strconv.ParseBool(*opt.Value); err == nil {
						c.phaseSwitching = val
					}

				case ocpp.KeyMaxChargingProfilesInstalled:
					if val, err := strconv.Atoi(*opt.Value); err == nil {
						c.chargingProfileId = val
					}

				case ocpp.KeyMeterValuesSampledData:
					if opt.Readonly {
						meterValuesSampledDataMaxLength = 0
					}
					meterValuesSampledData = *opt.Value

				case ocpp.KeyMeterValuesSampledDataMaxLength:
					if val, err := strconv.Atoi(*opt.Value); err == nil {
						meterValuesSampledDataMaxLength = val
					}

				case ocpp.KeyNumberOfConnectors:
					var val int
					if val, err = strconv.Atoi(*opt.Value); err == nil && connector > val {
						err = fmt.Errorf("connector %d exceeds max available connectors: %d", connector, val)
					}

				case ocpp.KeySupportedFeatureProfiles:
					if !c.hasProperty(*opt.Value, smartcharging.ProfileName) {
						c.log.WARN.Printf("the required SmartCharging feature profile is not indicated as supported")
					}
					// correct the availability assumption of RemoteTrigger only in case of a valid looking FeatureProfile list
					if c.hasProperty(*opt.Value, core.ProfileName) {
						c.hasRemoteTriggerFeature = c.hasProperty(*opt.Value, remotetrigger.ProfileName)
					}

				// vendor-specific keys
				case ocpp.KeyAlfenPlugAndChargeIdentifier:
					if c.idtag == defaultIdTag {
						c.idtag = *opt.Value
						c.log.DEBUG.Printf("overriding default `idTag` with Alfen-specific value: %s", c.idtag)
					}

				case ocpp.KeyEvBoxSupportedMeasurands:
					if meterValues == "" {
						meterValues = *opt.Value
					}
				}

				if err != nil {
					break
				}
			}
		}

		rc <- err
	}, nil)

	if err := c.wait(err, rc); err != nil {
		return nil, err
	}

	// see who's there
	if c.hasRemoteTriggerFeature {
		// CP
		if err := ocpp.Instance().TriggerMessageRequest(cp.ID(), core.BootNotificationFeatureName); err != nil {
			c.log.DEBUG.Printf("failed triggering BootNotification: %v", err)
		}

		select {
		case <-time.After(timeout):
			c.log.DEBUG.Printf("BootNotification timeout")
		case res := <-cp.BootNotificationRequest():
			if res != nil {
				c.bootNotification = res
			}
		}
	}

	// autodetect measurands
	if meterValues == "" && meterValuesSampledDataMaxLength > 0 {
		sampledMeasurands := c.tryMeasurands(desiredMeasurands, ocpp.KeyMeterValuesSampledData)
		meterValues = strings.Join(sampledMeasurands[:min(len(sampledMeasurands), meterValuesSampledDataMaxLength)], ",")
	}

	// configure measurands
	if meterValues != "" {
		// CP
		if err := c.configure(ocpp.KeyMeterValuesSampledData, meterValues); err == nil {
			meterValuesSampledData = meterValues
		}
	}

	c.meterValuesSample = meterValuesSampledData

	// trigger initial meter values
	if c.hasRemoteTriggerFeature {
		// CP
		if err := conn.TriggerMessageRequest(core.MeterValuesFeatureName); err == nil {
			// wait for meter values
			select {
			case <-time.After(timeout):
				c.log.WARN.Println("meter timeout")
			case <-c.conn.MeterSampled():
			}
		}
	}

	// configure sample rate
	if meterInterval > 0 {
		// CP
		if err := c.configure(ocpp.KeyMeterValueSampleInterval, strconv.Itoa(int(meterInterval.Seconds()))); err != nil {
			c.log.WARN.Printf("failed configuring MeterValueSampleInterval: %v", err)
		}
	}

	if c.hasRemoteTriggerFeature {
		// CP
		go conn.WatchDog(10 * time.Second)
	}

	// configure ping interval
	// CP
	c.configure(ocpp.KeyWebSocketPingInterval, "30")

	// CONN
	if c.hasRemoteTriggerFeature {
		if err := conn.TriggerMessageRequest(core.StatusNotificationFeatureName); err != nil {
			c.log.DEBUG.Printf("failed triggering StatusNotification: %v", err)
		}
	}

	return c, conn.Initialized()
}

// Connector returns the connector instance
func (c *OCPP) Connector() *ocpp.Connector {
	return c.conn
}

// hasMeasurement checks if meterValuesSample contains given measurement
func (c *OCPP) hasMeasurement(val types.Measurand) bool {
	return c.hasProperty(c.meterValuesSample, string(val))
}

// hasProperty checks if comma-separated string contains given string ignoring whitespaces
func (c *OCPP) hasProperty(props string, prop string) bool {
	return slices.ContainsFunc(strings.Split(props, ","), func(s string) bool {
		return strings.HasPrefix(strings.ReplaceAll(s, " ", ""), prop)
	})
}

func (c *OCPP) effectiveIdTag() string {
	if idtag := c.conn.IdTag(); idtag != "" {
		return idtag
	}
	return c.idtag
}

func (c *OCPP) tryMeasurands(measurands string, key string) []string {
	var accepted []string
	for _, m := range strings.Split(measurands, ",") {
		if err := c.configure(key, m); err == nil {
			accepted = append(accepted, m)
		}
	}
	return accepted
}

// configure updates CP configuration
func (c *OCPP) configure(key, val string) error {
	rc := make(chan error, 1)

	err := ocpp.Instance().ChangeConfiguration(c.conn.ChargePoint().ID(), func(resp *core.ChangeConfigurationConfirmation, err error) {
		if err == nil && resp != nil && resp.Status != core.ConfigurationStatusAccepted {
			rc <- fmt.Errorf("ChangeConfiguration failed: %s", resp.Status)
		}

		rc <- err
	}, key, val)

	return c.wait(err, rc)
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
	if c.hasMeasurement(types.MeasurandCurrentOffered) {
		if v, err := c.conn.GetMaxCurrent(); err == nil {
			return v > 0, nil
		}
	}
	if c.hasMeasurement(types.MeasurandPowerOffered) {
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
	err := ocpp.Instance().RemoteStartTransaction(c.conn.ChargePoint().ID(), func(resp *core.RemoteStartTransactionConfirmation, err error) {
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
	err := ocpp.Instance().SetChargingProfile(c.conn.ChargePoint().ID(), func(resp *smartcharging.SetChargingProfileConfirmation, err error) {
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
	err := ocpp.Instance().GetCompositeSchedule(c.conn.ChargePoint().ID(), func(resp *smartcharging.GetCompositeScheduleConfirmation, err error) {
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
	if c.chargingRateUnit == types.ChargingRateUnitWatts {
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

	return &types.ChargingProfile{
		ChargingProfileId:      c.chargingProfileId,
		StackLevel:             c.stackLevel,
		ChargingProfilePurpose: types.ChargingProfilePurposeTxDefaultProfile,
		ChargingProfileKind:    types.ChargingProfileKindAbsolute,
		ChargingSchedule: &types.ChargingSchedule{
			StartSchedule:          types.Now(),
			ChargingRateUnit:       c.chargingRateUnit,
			ChargingSchedulePeriod: []types.ChargingSchedulePeriod{period},
		},
	}
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
	fmt.Printf("\tCharge Point ID: %s\n", c.conn.ChargePoint().ID())

	if c.bootNotification != nil {
		fmt.Printf("\tBoot Notification:\n")
		fmt.Printf("\t\tChargePointVendor: %s\n", c.bootNotification.ChargePointVendor)
		fmt.Printf("\t\tChargePointModel: %s\n", c.bootNotification.ChargePointModel)
		fmt.Printf("\t\tChargePointSerialNumber: %s\n", c.bootNotification.ChargePointSerialNumber)
		fmt.Printf("\t\tFirmwareVersion: %s\n", c.bootNotification.FirmwareVersion)
	}

	fmt.Printf("\tConfiguration:\n")
	rc := make(chan error, 1)
	err := ocpp.Instance().GetConfiguration(c.conn.ChargePoint().ID(), func(resp *core.GetConfigurationConfirmation, err error) {
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
