package charger

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/samber/lo"
)

const statusTimeout = 30 * time.Second

// OCPP charger implementation
type OCPP struct {
	log                     *util.Logger
	cp                      *ocpp.CP
	connector               int
	idtag                   string
	phases                  int
	current                 float64
	meterValuesSample       string
	phaseSwitchingSupported bool
}

func init() {
	registry.Add("ocpp", NewOCPPFromConfig)
}

// NewOCPPFromConfig creates a OCPP charger from generic config
func NewOCPPFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		StationId     string
		IdTag         string
		Connector     int
		Meter         interface{} // TODO deprecated
		MeterInterval time.Duration
		MeterValues   string
		InitialReset  core.ResetType
		Timeout       time.Duration
	}{
		Connector: 1,
		Timeout:   time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	switch cc.InitialReset {
	case
		"",
		core.ResetTypeSoft,
		core.ResetTypeHard:
	default:
		return nil, fmt.Errorf("unknown configuration option detected for reset: %s", cc.InitialReset)
	}

	c, err := NewOCPP(cc.StationId, cc.Connector, cc.IdTag, cc.MeterValues, cc.MeterInterval, cc.InitialReset, cc.Timeout)
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

	return decorateOCPP(c, powerG, totalEnergyG, currentsG), nil
}

//go:generate go run ../cmd/tools/decorate.go -f decorateOCPP -b *OCPP -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)"

// NewOCPP creates OCPP charger
func NewOCPP(id string, connector int, idtag string, meterValues string, meterInterval time.Duration, initialReset core.ResetType, timeout time.Duration) (*OCPP, error) {
	unit := "ocpp"
	if id != "" {
		unit = id
	}
	log := util.NewLogger(unit)

	cp := ocpp.NewChargePoint(log, id, timeout)
	if err := ocpp.Instance().Register(id, cp); err != nil {
		return nil, err
	}

	c := &OCPP{
		log:       log,
		cp:        cp,
		connector: connector,
		idtag:     idtag,
	}

	c.log.DEBUG.Printf("waiting for chargepoint: %v", timeout)

	select {
	case <-time.After(timeout):
		return nil, api.ErrTimeout
	case <-cp.HasConnected():
	}

	// see who's there
	ocpp.Instance().TriggerMessageRequest(cp.ID(), core.BootNotificationFeatureName)

	var (
		rc                  = make(chan error, 1)
		options             []core.ConfigurationKey
		meterSampleInterval time.Duration
	)

	// configured id may be empty, use registered id below
	err := ocpp.Instance().GetConfiguration(cp.ID(), func(resp *core.GetConfigurationConfirmation, err error) {
		if err != nil {
			rc <- err
			return
		}

		options = resp.ConfigurationKey

		// sort config options for printing
		sort.Slice(options, func(i, j int) bool {
			return options[i].Key < options[j].Key
		})

		rw := map[bool]string{false: "r/w", true: "r/o"}

		for _, opt := range options {
			c.log.TRACE.Printf("%s (%s): %s", opt.Key, rw[opt.Readonly], *opt.Value)

			switch opt.Key {
			case ocpp.KeyMeterValuesSampledData:
				c.meterValuesSample = *opt.Value

			case ocpp.KeyMeterValueSampleInterval:
				meterValuesSampleInterval, err := strconv.Atoi(*opt.Value)
				if err != nil {
					rc <- err
					return
				}
				meterSampleInterval = time.Duration(meterValuesSampleInterval) * time.Second

			case string(ocpp.KeyConnectorSwitch3to1PhaseSupported):
				// Detection of 1 phase charging/switching support
				b, err := strconv.ParseBool(*opt.Value)
				if err != nil {
					rc <- err
					return
				}

				c.phaseSwitchingSupported = b
			}
		}

		rc <- nil
	}, []string{})

	if err := c.wait(err, rc); err != nil {
		return nil, err
	}

	if err := cp.DetectCapabilities(options); err != nil {
		return nil, err
	}

	// check supported connectors of charge point
	if supported := cp.GetNumberOfSupportedConnectors(); c.connector > supported {
		return nil, fmt.Errorf("configured connector is not available, max available connectors %d", supported)
	}

	if meterValues != "" {
		if err := c.configure(ocpp.KeyMeterValuesSampledData, meterValues); err != nil {
			return nil, err
		}

		// configuration activated
		c.meterValuesSample = meterValues
	}

	// get initial meter values and configure sample rate
	if c.hasMeasurement("Power.Active.Import") || c.hasMeasurement("Energy.Active.Import.Register") {
		ocpp.Instance().TriggerMessageRequest(cp.ID(), core.MeterValuesFeatureName)

		if meterSampleInterval > meterInterval && meterInterval > 0 {
			if err := c.configure(ocpp.KeyMeterValueSampleInterval, strconv.Itoa(int(meterInterval.Seconds()))); err != nil {
				return nil, err
			}

			// HACK: setup watchdog for meter values if not happy with config
			c.log.DEBUG.Println("enabling meter watchdog")
			cp.WatchDog(meterInterval)
		}
	}

	if initialReset != "" {
		t := core.ResetTypeSoft
		if initialReset == core.ResetTypeHard {
			t = core.ResetTypeHard
		}

		ocpp.Instance().TriggerResetRequest(cp.ID(), t)
	}

	// request initial status
	_ = cp.Initialized(statusTimeout)

	// TODO: check for running transaction

	return c, nil
}

// hasMeasurement checks if meterValuesSample contains given measurement
func (c *OCPP) hasMeasurement(val types.Measurand) bool {
	return lo.Contains(strings.Split(c.meterValuesSample, ","), string(val))
}

// configure updates CP configuration
func (c *OCPP) configure(key, val string) error {
	rc := make(chan error, 1)

	err := ocpp.Instance().ChangeConfiguration(c.cp.ID(), func(resp *core.ChangeConfigurationConfirmation, err error) {
		c.log.TRACE.Printf("ChangeConfiguration: %v", resp)

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
		case <-time.After(request.Timeout):
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
	return c.cp.TransactionID() > 0, nil
}

// Enable implements the api.Charger interface
func (c *OCPP) Enable(enable bool) error {
	var err error
	rc := make(chan error, 1)

	if enable {
		err = ocpp.Instance().RemoteStartTransaction(c.cp.ID(), func(resp *core.RemoteStartTransactionConfirmation, err error) {
			c.log.TRACE.Printf("RemoteStartTransaction: %+v", resp)

			if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
				err = errors.New(string(resp.Status))
			}

			rc <- err
		}, c.idtag, func(request *core.RemoteStartTransactionRequest) {
			request.ConnectorId = &c.connector
		})
	} else {
		err = ocpp.Instance().RemoteStopTransaction(c.cp.ID(), func(resp *core.RemoteStopTransactionConfirmation, err error) {
			c.log.TRACE.Printf("RemoteStopTransaction: %+v", resp)

			if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
				err = errors.New(string(resp.Status))
			}

			rc <- err
		}, c.cp.TransactionID())
	}

	return c.wait(err, rc)
}

func (c *OCPP) setChargingProfile(connectorid int, profile *types.ChargingProfile) error {
	c.log.TRACE.Printf("SetChargingProfileRequest: %+v (%+v)", profile, *profile.ChargingSchedule)

	rc := make(chan error, 1)
	err := ocpp.Instance().SetChargingProfile(c.cp.ID(), func(resp *smartcharging.SetChargingProfileConfirmation, err error) {
		c.log.TRACE.Printf("SetChargingProfile: %+v", resp)
		if err == nil && resp != nil && resp.Status != smartcharging.ChargingProfileStatusAccepted {
			err = errors.New(string(resp.Status))
		}

		rc <- err
	}, connectorid, profile)

	return c.wait(err, rc)
}

// setPeriod sets a single charging schedule period with given current and phases
func (c *OCPP) setPeriod(current float64, phases int) error {
	period := types.NewChargingSchedulePeriod(0, current)

	c.log.TRACE.Printf("current phases: %d, current current: %f", phases, current)
	if phases > 0 {
		period.NumberPhases = &phases
	}

	// connectorID: 0 - profile will be applied to all connectors
	err := c.setChargingProfile(0, getMaxChargingProfile(period))
	if err != nil {
		err = fmt.Errorf("failed to set charging profile: %w", err)
	}

	return err
}

func getMaxChargingProfile(period types.ChargingSchedulePeriod) *types.ChargingProfile {
	return &types.ChargingProfile{
		ChargingProfileId:      1,
		StackLevel:             1,
		ChargingProfilePurpose: types.ChargingProfilePurposeChargePointMaxProfile,
		ChargingProfileKind:    types.ChargingProfileKindAbsolute,
		ChargingSchedule: &types.ChargingSchedule{
			StartSchedule:          types.NewDateTime(time.Now().Add(-1 * time.Hour)),
			ChargingRateUnit:       types.ChargingRateUnitAmperes,
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
	err := c.setPeriod(current, c.phases)
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

// Currents implements the api.MeterCurrent interface
func (c *OCPP) currents() (float64, float64, float64, error) {
	return c.cp.Currents()
}

// // TODO: Phases1p3p implements the api.PhaseSwitcher interface
// func (c *OCPP) Phases1p3p(phases int) error {
// 	if !c.phaseSwitchingSupported {
// 		return fmt.Errorf("phase switching is not supported by the charger")
// 	}

// 	err := c.setPeriod(c.current, phases)
// 	if err == nil {
// 		c.phases = phases
// 	}

// 	return err
// }

// // Identify implements the api.Identifier interface
// func (c *OCPP) Identify() (string, error) {
// 	return "", errors.New("not implemented")
// }
