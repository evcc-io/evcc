package charger

import (
	"errors"
	"fmt"
	"math"
	"sort"
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
	"github.com/samber/lo"
)

// OCPP charger implementation
type OCPP struct {
	log               *util.Logger
	cp                *ocpp.CP
	connector         int
	idtag             string
	enabled           bool
	phases            int
	current           float64
	meterValuesSample string
	timeout           time.Duration
	phaseSwitching    bool
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
		boot, noConfig,
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

	// removed to avoid sending phases when updating currents
	var phasesS func(int) error
	// if c.phaseSwitching {
	// 	phasesS = c.phases1p3p
	// }
	_ = c.phases1p3p

	return decorateOCPP(c, powerG, totalEnergyG, currentsG, phasesS), nil
}

// go:generate go run ../cmd/tools/decorate.go -f decorateOCPP -b *OCPP -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewOCPP creates OCPP charger
func NewOCPP(id string, connector int, idtag string,
	meterValues string, meterInterval time.Duration,
	boot, noConfig bool,
	connectTimeout, timeout time.Duration,
	chargingRateUnit string,
) (*OCPP, error) {
	unit := "ocpp"
	if id != "" {
		unit = id
	}
	log := util.NewLogger(unit)

	cp := ocpp.NewChargePointConnector(log, id, connector, timeout)
	if err := ocpp.Instance().Register(id, cp); err != nil {
		return nil, err
	}

	c := &OCPP{
		log:       log,
		cp:        cp,
		connector: connector,
		idtag:     idtag,
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
		ocpp.Instance().TriggerMessageRequest(cp.ID(), core.BootNotificationFeatureName)
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
		err := ocpp.Instance().GetConfiguration(cp.ID(), func(resp *core.GetConfigurationConfirmation, err error) {
			if err == nil {
				// log unsupported configuration keys
				if len(resp.UnknownKey) > 0 {
					c.log.ERROR.Printf("unsupported keys: %v", sort.StringSlice(resp.UnknownKey))
				}

				// sort configuration keys for printing
				sort.Slice(resp.ConfigurationKey, func(i, j int) bool {
					return resp.ConfigurationKey[i].Key < resp.ConfigurationKey[j].Key
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
						if val, err = strconv.Atoi(*opt.Value); err == nil && c.connector > val {
							err = fmt.Errorf("connector %d exceeds max available connectors: %d", c.connector, val)
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
		ocpp.Instance().TriggerMeterValuesRequest(cp.ID(), cp.Connector())

		if meterInterval > 0 && meterInterval != meterSampleInterval {
			if err := c.configure(ocpp.KeyMeterValueSampleInterval, strconv.Itoa(int(meterInterval.Seconds()))); err != nil {
				return nil, err
			}
		}

		// HACK: setup watchdog for meter values if not happy with config
		if meterInterval > 0 {
			c.log.DEBUG.Println("enabling meter watchdog")
			go cp.WatchDog(meterInterval)
		}
	}

	// TODO: check for running transaction

	return c, cp.Initialized()
}

// hasMeasurement checks if meterValuesSample contains given measurement
func (c *OCPP) hasMeasurement(val types.Measurand) bool {
	return lo.Contains(strings.Split(c.meterValuesSample, ","), string(val))
}

// configure updates CP configuration
func (c *OCPP) configure(key, val string) error {
	rc := make(chan error, 1)

	err := ocpp.Instance().ChangeConfiguration(c.cp.ID(), func(resp *core.ChangeConfigurationConfirmation, err error) {
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
	return c.cp.Status()
}

// Enabled implements the api.Charger interface
func (c *OCPP) Enabled() (bool, error) {
	return c.enabled, nil
}

// Enable implements the api.Charger interface
func (c *OCPP) Enable(enable bool) (err error) {
	rc := make(chan error, 1)
	txn, err := c.cp.TransactionID()

	defer func() {
		if err == nil {
			c.enabled = enable
		}
	}()

	if enable {
		if txn > 0 {
			return errors.New("cannot enable: transaction already running")
		}

		err = ocpp.Instance().RemoteStartTransaction(c.cp.ID(), func(resp *core.RemoteStartTransactionConfirmation, err error) {
			if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
				err = errors.New(string(resp.Status))
			}

			rc <- err
		}, c.idtag, func(request *core.RemoteStartTransactionRequest) {
			request.ConnectorId = &c.connector
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

		err = ocpp.Instance().RemoteStopTransaction(c.cp.ID(), func(resp *core.RemoteStopTransactionConfirmation, err error) {
			if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
				err = errors.New(string(resp.Status))
			}

			rc <- err
		}, txn)
	}

	return c.wait(err, rc)
}

func (c *OCPP) setChargingProfile(connectorId int, profile *types.ChargingProfile) error {
	rc := make(chan error, 1)
	err := ocpp.Instance().SetChargingProfile(c.cp.ID(), func(resp *smartcharging.SetChargingProfileConfirmation, err error) {
		if err == nil && resp != nil && resp.Status != smartcharging.ChargingProfileStatusAccepted {
			err = errors.New(string(resp.Status))
		}

		rc <- err
	}, connectorId, profile)

	return c.wait(err, rc)
}

// updatePeriod sets a single charging schedule period with given current
func (c *OCPP) updatePeriod(current float64) error {
	// current period can only be updated if transaction is active
	if enabled, err := c.Enabled(); err != nil || !enabled {
		return err
	}

	txn, err := c.cp.TransactionID()
	if err != nil {
		return err
	}

	current = math.Trunc(10*current) / 10

	err = c.setChargingProfile(c.connector, c.getTxChargingProfile(current, txn))
	if err != nil {
		err = fmt.Errorf("set charging profile: %w", err)
	}

	return err
}

func (c *OCPP) getTxChargingProfile(current float64, transactionId int) *types.ChargingProfile {
	period := types.NewChargingSchedulePeriod(0, current)
	if c.chargingRateUnit == types.ChargingRateUnitWatts {
		var phases int
		// get (expectedly) active phases from loadpoint
		if c.lp != nil {
			phases = c.lp.GetPhases()
		}
		if phases == 0 {
			phases = 3
		}
		period = types.NewChargingSchedulePeriod(0, math.Trunc(230.0*current*float64(phases)))
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
	return c.cp.CurrentPower()
}

// TotalEnergy implements the api.MeterTotal interface
func (c *OCPP) totalEnergy() (float64, error) {
	return c.cp.TotalEnergy()
}

// Currents implements the api.PhaseCurrents interface
func (c *OCPP) currents() (float64, float64, float64, error) {
	return c.cp.Currents()
}

// Phases1p3p implements the api.PhaseSwitcher interface
func (c *OCPP) phases1p3p(phases int) error {
	c.phases = phases

	// NOTE: this will currently _never_ do anything since
	// loadpoint disabled the charger before switching so
	// updatePeriod will short-circuit
	return c.updatePeriod(c.current)
}

// // Identify implements the api.Identifier interface
// Unless charger uses vehicle ID as idTag in authorize.req it is not possible to implement this in ocpp1.6
// func (c *OCPP) Identify() (string, error) {
// 	return "", errors.New("not implemented")
// }

// LoadpointControl implements loadpoint.Controller
func (c *OCPP) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}
