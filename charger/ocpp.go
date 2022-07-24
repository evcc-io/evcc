package charger

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// OCPP charger implementation
type OCPP struct {
	log                     *util.Logger
	cp                      *ocpp.CP
	id                      string
	connector               int
	idtag                   string
	phases                  int
	current                 float64
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
		Meter         bool
		MeterInterval time.Duration
		InitialReset  core.ResetType
	}{
		Connector: 1,
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

	ocpp, err := NewOCPP(cc.StationId, cc.Connector, cc.IdTag, cc.Meter, cc.MeterInterval, cc.InitialReset)
	if err != nil {
		return ocpp, err
	}

	var (
		meter        func() (float64, error)
		meterCurrent func() (float64, float64, float64, error)
		chargeRater  func() (float64, error)
	)

	if cc.Meter {
		meter = ocpp.currentPower
		meterCurrent = ocpp.currents
		chargeRater = ocpp.chargedEnergy
	}

	return decorateOCPP(ocpp, meter, meterCurrent, chargeRater), nil
}

//go:generate go run ../cmd/tools/decorate.go -f decorateOCPP -b *OCPP -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)" -t "api.ChargeRater,ChargedEnergy,func() (float64, error)"

// NewOCPP creates OCPP charger
func NewOCPP(id string, connector int, idtag string, hasMeter bool, meterInterval time.Duration, initialReset core.ResetType) (*OCPP, error) {
	cp := ocpp.Instance().Register(id, hasMeter)

	logstr := "-charger"
	if id != "" {
		logstr = fmt.Sprintf("-%s", id)
	}

	c := &OCPP{
		log:       util.NewLogger(fmt.Sprintf("ocpp%s:%d", logstr, connector)),
		cp:        cp,
		id:        id,
		connector: connector,
		idtag:     idtag,
	}

	if err := cp.Boot(); err != nil {
		return nil, err
	}

	var (
		rc                           = make(chan error, 1)
		options                      []core.ConfigurationKey
		meterValuesSampledDataString string
		meterValuesSampleInterval    string
	)

	err := ocpp.Instance().CS().GetConfiguration(id, func(resp *core.GetConfigurationConfirmation, err error) {
		options = resp.ConfigurationKey

		for _, opt := range options {
			c.log.TRACE.Printf("%s (%t): %s", opt.Key, opt.Readonly, *opt.Value)
			switch opt.Key {
			case ocpp.KeyMeterValuesSampledData:
				meterValuesSampledDataString = *opt.Value
			case ocpp.KeyMeterValueSampleInterval:
				meterValuesSampleInterval = *opt.Value
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

		rc <- err
	}, []string{})

	if err := c.wait(err, rc); err != nil {
		return nil, err
	}

	if err := cp.DetectCapabilities(options); err != nil {
		return nil, err
	}

	{ // Check supported connectors of charge point
		supported := cp.GetNumberOfSupportedConnectors()
		if c.connector > supported {
			return nil, fmt.Errorf("configured connector is not available, max available connectors %d", supported)
		}
	}

	if hasMeter {
		if meterValuesSampledDataString != "Current.Import,Current.Offered,Energy.Active.Import.Register,Power.Active.Import,Temperature" {
			c.log.TRACE.Printf("Current values \n\t%s != \n\t%+v", ocpp.ValuePreferedMeterValuesSampleData, meterValuesSampledDataString)
			rc = make(chan error, 1)
			err = ocpp.Instance().CS().ChangeConfiguration(id, func(resp *core.ChangeConfigurationConfirmation, err error) {
				c.log.TRACE.Printf("ChangeMeterConfigurationRequest %T: %+v", resp, resp)

				if resp.Status == core.ConfigurationStatusRejected {
					rc <- fmt.Errorf("configuration change rejected")
				}

				rc <- err
			}, ocpp.KeyMeterValuesSampledData, ocpp.ValuePreferedMeterValuesSampleData)

			if err := c.wait(err, rc); err != nil {
				return nil, err
			}
		}

		{
			intervalStr := fmt.Sprintf("%d", int(meterInterval.Seconds()))
			if meterValuesSampleInterval != intervalStr {
				rc = make(chan error, 1)

				err := ocpp.Instance().CS().ChangeConfiguration(id, func(resp *core.ChangeConfigurationConfirmation, err error) {
					c.log.TRACE.Printf("ChangeSampleMeterValueInterval %T: %v", resp, resp)

					if resp.Status == core.ConfigurationStatusRejected {
						rc <- fmt.Errorf("configuration of meter interval rejected: %w", err)
					}
					rc <- err
				}, ocpp.KeyMeterValueSampleInterval, intervalStr)

				if err := c.wait(err, rc); err != nil {
					return nil, err
				}
			}
		}

		// get initial meter values
		if hasMeter {
			ocpp.Instance().TriggerMeterValueRequest(cp)
		}
	}

	if initialReset != "" {
		t := core.ResetTypeSoft
		if initialReset == core.ResetTypeHard {
			t = core.ResetTypeHard
		}

		ocpp.Instance().TriggerResetRequest(cp, t)
	}

	// TODO: check for running transaction

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *OCPP) Enabled() (bool, error) {
	current, err := c.cp.Status()
	if current == api.StatusC {
		return true, err
	}

	return false, err
}

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

// Enable implements the api.Charger interface
func (c *OCPP) Enable(enable bool) error {
	var err error
	rc := make(chan error, 1)

	if enable {
		err = ocpp.Instance().CS().RemoteStartTransaction(c.id, func(resp *core.RemoteStartTransactionConfirmation, err error) {
			c.log.TRACE.Printf("RemoteStartTransaction %T: %+v", resp, resp)

			if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
				err = errors.New(string(resp.Status))
			}

			rc <- err
		}, c.idtag, func(request *core.RemoteStartTransactionRequest) {
			request.ConnectorId = &c.connector
		})
	} else {
		err = ocpp.Instance().CS().RemoteStopTransaction(c.id, func(resp *core.RemoteStopTransactionConfirmation, err error) {
			c.log.TRACE.Printf("RemoteStopTransaction %T: %+v", resp, resp)

			if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
				err = errors.New(string(resp.Status))
			}

			rc <- err
		}, c.cp.TransactionID())
	}

	return c.wait(err, rc)
}

func (c *OCPP) setChargingProfile(connectorid int, profile *types.ChargingProfile) error {
	c.log.TRACE.Printf("SetChargingProfileRequest %T: %+v", profile, profile)
	c.log.TRACE.Printf("SetChargingProfileRequest %T: %+v", profile.ChargingSchedule, profile.ChargingSchedule)

	rc := make(chan error, 1)
	err := ocpp.Instance().CS().SetChargingProfile(c.id, func(resp *smartcharging.SetChargingProfileConfirmation, err error) {
		c.log.TRACE.Printf("SetChargingProfileResponse %T: %+v", resp, resp)
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
	err := c.setChargingProfile(0, getMaxCharginProfile(period))
	if err != nil {
		c.log.TRACE.Printf("failed to set charging profile: %s", err)
	}

	return err
}

func getMaxCharginProfile(period types.ChargingSchedulePeriod) *types.ChargingProfile {
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

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *OCPP) MaxCurrentMillis(current float64) error {
	err := c.setPeriod(current, c.phases)
	if err == nil {
		c.current = current
	}
	return err
}

// Status implements the api.Charger interface
func (c *OCPP) Status() (api.ChargeStatus, error) {
	return c.cp.Status()
}

// TODO: Phases1p3p implements the api.ChargePhases interface
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

// CurrentPower implements the api.Meter interface
func (c *OCPP) currentPower() (float64, error) {
	return c.cp.CurrentPower()
}

// ChargedEnergy implements the api.ChargeRater interface
func (c *OCPP) chargedEnergy() (float64, error) {
	return c.cp.ChargedEnergy()
}

// Currents implements the api.MeterCurrent interface
func (c *OCPP) currents() (float64, float64, float64, error) {
	return c.cp.Currents()
}

// // Identify implements the api.Identifier interface
// func (c *OCPP) Identify() (string, error) {
// 	return "", errors.New("not implemented")
// }
