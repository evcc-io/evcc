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
	enabled           bool
	phases            int
	current           float64
	meterValuesSample string
	timeout           time.Duration
	phaseSwitching    bool
	autoStart         bool
	chargingRateUnit  types.ChargingRateUnitType
	lp                loadpoint.API
}

const defaultIdTag = "evcc"

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
		AutoStart        bool
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
		boot, noConfig, cc.AutoStart,
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

	var phasesS func(int) error
	if c.phaseSwitching {
		phasesS = c.phases1p3p
	}

	return decorateOCPP(c, powerG, totalEnergyG, currentsG, phasesS), nil
}

//go:generate go run ../cmd/tools/decorate.go -f decorateOCPP -b *OCPP -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewOCPP creates OCPP charger
func NewOCPP(id string, connector int, idtag string,
	meterValues string, meterInterval time.Duration,
	boot, noConfig, autoStart bool,
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
		log:       log,
		conn:      conn,
		idtag:     idtag,
		autoStart: autoStart,
		timeout:   timeout,
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
						if *opt.Value == "W" {
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
	if err == nil {
		select {
		case err = <-rc:
			close(rc)
		case <-time.After(c.timeout):
			err = api.ErrTimeout
		}
	}
	return err
}

// Status implements the api.Charger interface
func (c *OCPP) Status() (api.ChargeStatus, error) {
	return c.conn.Status()
}

// Enabled implements the api.Charger interface
func (c *OCPP) Enabled() (bool, error) {
	return c.enabled, nil
}

func (c *OCPP) Enable(enable bool) error {
	var err error

	if c.autoStart {
		err = c.enableAutostart(enable)
	} else {
		err = c.enableRemote(enable)
	}

	if err == nil {
		c.enabled = enable
	}

	return err
}

// enableAutostart enables auto-started session
func (c *OCPP) enableAutostart(enable bool) error {
	var current float64
	if enable {
		current = c.current
	}

	return c.updatePeriod(current)
}

// enableRemote enables session by using RemoteStart/Stop
func (c *OCPP) enableRemote(enable bool) error {
	txn, err := c.conn.TransactionID()
	if err != nil {
		return err
	}

	rc := make(chan error, 1)
	if enable {
		if txn > 0 {
			// we have the transaction id, treat as enabled
			return nil
		}

		err = ocpp.Instance().RemoteStartTransaction(c.conn.ChargePoint().ID(), func(resp *core.RemoteStartTransactionConfirmation, err error) {
			if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
				err = errors.New(string(resp.Status))
			}

			rc <- err
		}, c.effectiveIdTag(), func(request *core.RemoteStartTransactionRequest) {
			connector := c.conn.ID()
			request.ConnectorId = &connector
			request.ChargingProfile = c.getTxChargingProfile(c.current, 0)
		})
	} else {
		// if no transaction is running, the vehicle may have stopped it (which is ok) or an unknown transaction is running
		if txn == 0 {
			// we cannot tell if a transaction is really running, so we check the status
			status, err := c.Status()
			if err != nil {
				return err
			}
			if status == api.StatusC {
				return errors.New("cannot disable: unknown transaction running")
			}

			return nil
		}

		err = ocpp.Instance().RemoteStopTransaction(c.conn.ChargePoint().ID(), func(resp *core.RemoteStopTransactionConfirmation, err error) {
			if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
				err = errors.New(string(resp.Status))
			}

			rc <- err
		}, txn)
	}

	return c.wait(err, rc)
}

func (c *OCPP) setChargingProfile(profile *types.ChargingProfile) error {
	connector := c.conn.ID()

	rc := make(chan error, 1)
	err := ocpp.Instance().SetChargingProfile(c.conn.ChargePoint().ID(), func(resp *smartcharging.SetChargingProfileConfirmation, err error) {
		if err == nil && resp != nil && resp.Status != smartcharging.ChargingProfileStatusAccepted {
			err = errors.New(string(resp.Status))
		}

		rc <- err
	}, connector, profile)

	return c.wait(err, rc)
}

// updatePeriod sets a single charging schedule period with given current
func (c *OCPP) updatePeriod(current float64) error {
	// current period can only be updated if transaction is active
	if enabled, err := c.Enabled(); err != nil || !enabled {
		return err
	}

	txn, err := c.conn.TransactionID()
	if err != nil {
		return err
	}

	current = math.Trunc(10*current) / 10

	err = c.setChargingProfile(c.getTxChargingProfile(current, txn))
	if err != nil {
		err = fmt.Errorf("set charging profile: %w", err)
	}

	return err
}

func (c *OCPP) getTxChargingProfile(current float64, transactionId int) *types.ChargingProfile {
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
		ChargingProfileId:      1,
		TransactionId:          transactionId,
		StackLevel:             0,
		ChargingProfilePurpose: types.ChargingProfilePurposeTxProfile,
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
	err := c.updatePeriod(current)
	if err == nil {
		c.current = current
	}
	return err
}

// CurrentPower implements the api.Meter interface
func (c *OCPP) currentPower() (float64, error) {
	return c.conn.CurrentPower()
}

// TotalEnergy implements the api.MeterTotal interface
func (c *OCPP) totalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}

// Currents implements the api.PhaseCurrents interface
func (c *OCPP) currents() (float64, float64, float64, error) {
	return c.conn.Currents()
}

// Phases1p3p implements the api.PhaseSwitcher interface
func (c *OCPP) phases1p3p(phases int) error {
	c.phases = phases

	// NOTE: this will currently _never_ do anything since
	// loadpoint disabled the charger before switching so
	// updatePeriod will short-circuit
	return c.updatePeriod(c.current)
}

// Identify implements the api.Identifier interface
func (c *OCPP) Identify() (string, error) {
	return c.conn.IdTag(), nil
}

var _ loadpoint.Controller = (*OCPP)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *OCPP) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}
