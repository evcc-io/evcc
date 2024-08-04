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
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// OCPP charger implementation
type OCPP struct {
	log               *util.Logger
	conn              *ocpp.Connector
	idtag             string
	phases            int
	current           float64
	enabled           bool
	meterValuesSample string
	timeout           time.Duration
	phaseSwitching    bool
	remoteStart       bool
	chargingRateUnit  types.ChargingRateUnitType
	lp                loadpoint.API
	bootNotification  *core.BootNotificationRequest
}

const defaultIdTag = "evcc" // RemoteStartTransaction only

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
		ConnectTimeout   time.Duration
		Timeout          time.Duration
		BootNotification *bool
		GetConfiguration *bool
		ChargingRateUnit string
		AutoStart        bool // deprecated, to be removed
		NoStop           bool // deprecated, to be removed
		RemoteStart      bool
	}{
		Connector:        1,
		IdTag:            defaultIdTag,
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

	var powerG func() (float64, error)
	if c.hasMeasurement(types.MeasurandPowerActiveImport) {
		powerG = c.currentPower
	}

	var totalEnergyG func() (float64, error)
	if c.hasMeasurement(types.MeasurandEnergyActiveImportRegister) {
		totalEnergyG = c.totalEnergy
	}

	var currentsG func() (float64, float64, float64, error)
	if c.hasMeasurement(types.MeasurandCurrentImport + ".L3") {
		currentsG = c.currents
	}

	var voltagesG func() (float64, float64, float64, error)
	if c.hasMeasurement(types.MeasurandVoltage + ".L3") {
		voltagesG = c.voltages
	}

	var phasesS func(int) error
	if c.phaseSwitching {
		phasesS = c.phases1p3p
	}

	var socG func() (float64, error)
	if c.hasMeasurement(types.MeasurandSoC) {
		socG = c.soc
	}

	//var currentG func() (float64, error)
	//if c.hasMeasurement(types.MeasurandCurrentOffered) {
	//	currentG = c.getMaxCurrent
	//}

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
		timeout:     timeout,
	}

	c.log.DEBUG.Printf("waiting for chargepoint: %v", connectTimeout)

	select {
	case <-time.After(connectTimeout):
		return nil, api.ErrTimeout
	case <-cp.HasConnected():
	}

	// see who's there
	if boot {
		conn.TriggerMessageRequest(core.BootNotificationFeatureName)
		select {
		case <-time.After(timeout):
			c.log.WARN.Printf("boot notification timeout")
		case res := <-cp.BootNotificationRequest():
			c.bootNotification = res
		}
	}

	var (
		rc                  = make(chan error, 1)
		meterSampleInterval time.Duration
	)

	keys := []string{
		ocpp.KeyNumberOfConnectors,
		ocpp.KeyMeterValuesSampledData,
		ocpp.KeyMeterValueSampleInterval,
		ocpp.KeyConnectorSwitch3to1PhaseSupported,
		ocpp.KeyChargingScheduleAllowedChargingRateUnit,
	}
	_ = keys

	c.chargingRateUnit = types.ChargingRateUnitType(chargingRateUnit)

	// noConfig mode disables GetConfiguration
	if noConfig {
		c.meterValuesSample = meterValues
		if meterInterval == 0 {
			meterInterval = 10 * time.Second
		}
	} else {
		// fix timing issue in EVBox when switching OCPP protocol version
		time.Sleep(time.Second)

		err := ocpp.Instance().GetConfiguration(cp.ID(), func(resp *core.GetConfigurationConfirmation, err error) {
			if err == nil {
				// log unsupported configuration keys
				if len(resp.UnknownKey) > 0 {
					c.log.ERROR.Printf("unsupported keys: %v", resp.UnknownKey)
				}

				// sort configuration keys for printing
				slices.SortFunc(resp.ConfigurationKey, func(i, j core.ConfigurationKey) int {
					return cmp.Compare(i.Key, j.Key)
				})

				rw := map[bool]string{false: "r/w", true: "r/o"}

				for _, opt := range resp.ConfigurationKey {
					if opt.Value == nil {
						continue
					}

					c.log.TRACE.Printf("%s (%s): %s", opt.Key, rw[opt.Readonly], *opt.Value)

					switch opt.Key {
					case ocpp.KeyNumberOfConnectors:
						var val int
						if val, err = strconv.Atoi(*opt.Value); err == nil && connector > val {
							err = fmt.Errorf("connector %d exceeds max available connectors: %d", connector, val)
						}

					case ocpp.KeyMeterValuesSampledData:
						c.meterValuesSample = *opt.Value

					case ocpp.KeyMeterValueSampleInterval:
						var val int
						if val, err = strconv.Atoi(*opt.Value); err == nil {
							meterSampleInterval = time.Duration(val) * time.Second
						}

					case ocpp.KeyConnectorSwitch3to1PhaseSupported:
						var val bool
						if val, err = strconv.ParseBool(*opt.Value); err == nil {
							c.phaseSwitching = val
						}

					case ocpp.KeyAlfenPlugAndChargeIdentifier:
						if c.idtag == defaultIdTag {
							c.idtag = *opt.Value
							c.log.DEBUG.Printf("overriding default `idTag` with Alfen-specific value: %s", c.idtag)
						}

					case ocpp.KeyChargingScheduleAllowedChargingRateUnit:
						if *opt.Value == "W" || *opt.Value == "Power" {
							c.chargingRateUnit = types.ChargingRateUnitWatts
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
	}

	if meterValues != "" && meterValues != c.meterValuesSample {
		if err := c.configure(ocpp.KeyMeterValuesSampledData, meterValues); err != nil {
			return nil, err
		}

		// configuration activated
		c.meterValuesSample = meterValues
	}

	// get initial meter values and configure sample rate
	if c.hasMeasurement(types.MeasurandPowerActiveImport) || c.hasMeasurement(types.MeasurandEnergyActiveImportRegister) {
		conn.TriggerMessageRequest(core.MeterValuesFeatureName)

		// wait for meter values
		select {
		case <-time.After(timeout):
			c.log.WARN.Println("meter value timeout")
		case <-c.conn.MeterSampled():
		}

		if meterInterval > 0 && meterInterval != meterSampleInterval {
			if err := c.configure(ocpp.KeyMeterValueSampleInterval, strconv.Itoa(int(meterInterval.Seconds()))); err != nil {
				return nil, err
			}
		}

		// HACK: setup watchdog for meter values if not happy with config
		if meterInterval > 0 {
			c.log.DEBUG.Println("enabling meter watchdog")
			go conn.WatchDog(meterInterval)
		}
	}

	// TODO: check for running transaction

	return c, conn.Initialized()
}

// Connector returns the connector instance
func (c *OCPP) Connector() *ocpp.Connector {
	return c.conn
}

// hasMeasurement checks if meterValuesSample contains given measurement
func (c *OCPP) hasMeasurement(val types.Measurand) bool {
	return slices.Contains(strings.Split(c.meterValuesSample, ","), string(val))
}

func (c *OCPP) effectiveIdTag() string {
	if idtag := c.conn.IdTag(); idtag != "" {
		return idtag
	}
	return c.idtag
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
	if c.remoteStart {
		needtxn, err := c.conn.NeedsTransaction()
		if err != nil {
			return api.StatusNone, err
		}

		if needtxn {
			// lock the cable by starting remote transaction after vehicle connected
			if err := c.initTransaction(); err != nil {
				return api.StatusNone, err
			}
		}
	}

	return c.conn.Status()
}

// Enabled implements the api.Charger interface
func (c *OCPP) Enabled() (bool, error) {
	if s, err := c.conn.StatusOCPP(); err == nil {
		switch s {
		case core.ChargePointStatusSuspendedEVSE:
			return false, nil
		case core.ChargePointStatusCharging, core.ChargePointStatusSuspendedEV:
			return true, nil
		}
	}

	// fallback to the "offered" measurands
	if c.hasMeasurement(types.MeasurandCurrentOffered) {
		if v, err := c.getMaxCurrent(); err == nil {
			return v > 0, nil
		}
	}
	if c.hasMeasurement(types.MeasurandPowerOffered) {
		if v, err := c.getMaxPower(); err == nil {
			return v > 0, err
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

// getScheduleLimit querys the current or power limit the charge point is currently set to offer
func (c *OCPP) getScheduleLimit() (float64, error) {
	const duration int = 60 // duration of requested schedule in seconds

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
		ChargingProfileId:      0,
		StackLevel:             0,
		ChargingProfilePurpose: types.ChargingProfilePurposeTxDefaultProfile,
		ChargingProfileKind:    types.ChargingProfileKindRelative,
		ChargingSchedule: &types.ChargingSchedule{
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

// getMaxCurrent implements the api.CurrentGetter interface
func (c *OCPP) getMaxCurrent() (float64, error) {
	return c.conn.GetMaxCurrent()
}

// getMaxPower implements the api.PowerGetter interface
func (c *OCPP) getMaxPower() (float64, error) {
	return c.conn.GetMaxPower()
}

// currentPower implements the api.Meter interface
func (c *OCPP) currentPower() (float64, error) {
	return c.conn.CurrentPower()
}

// totalEnergy implements the api.MeterTotal interface
func (c *OCPP) totalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}

// currents implements the api.PhaseCurrents interface
func (c *OCPP) currents() (float64, float64, float64, error) {
	return c.conn.Currents()
}

// voltages implements the api.PhaseVoltages interface
func (c *OCPP) voltages() (float64, float64, float64, error) {
	return c.conn.Voltages()
}

// phases1p3p implements the api.PhaseSwitcher interface
func (c *OCPP) phases1p3p(phases int) error {
	c.phases = phases

	return c.setCurrent(c.current)
}

// soc implements the api.Battery interface
func (c *OCPP) soc() (float64, error) {
	return c.conn.Soc()
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
