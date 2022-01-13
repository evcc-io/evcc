package charger

import (
	"errors"
	"fmt"
	"sync"
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
	log       *util.Logger
	cp        *ocpp.CP
	id        string
	connector int
	idtag     string
	phases    int
	current   float64
}

func init() {
	registry.Add("ocpp", NewOCPPFromConfig)
}

// NewOCPPFromConfig creates a OCPP charger from generic config
func NewOCPPFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		StationId string
		IdTag     string
		Connector int
	}{
		Connector: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOCPP(cc.StationId, cc.Connector, cc.IdTag)
}

// NewOCPP creates OCPP charger
func NewOCPP(id string, connector int, idtag string) (*OCPP, error) {
	cp := ocpp.Instance().Register(id)
	c := &OCPP{
		log:       util.NewLogger(fmt.Sprintf("ocpp-%s:%d", id, connector)),
		cp:        cp,
		id:        id,
		connector: connector,
		idtag:     idtag,
	}

	if err := cp.Boot(); err != nil {
		return nil, err
	}

	wg := &sync.WaitGroup{}

	var options []core.ConfigurationKey

	wg.Add(1)
	if err := ocpp.Instance().CS().GetConfiguration(id, func(resp *core.GetConfigurationConfirmation, err error) {
		options = resp.ConfigurationKey
		wg.Done()
	}, []string{}); err != nil {
		return nil, err
	}

	wg.Wait()

	if err := cp.DetectCapabilities(options); err != nil {
		return nil, err
	}

	{ // Check supported connectors of charge point
		supported := cp.GetNumberOfSupportedConnectors()
		if c.connector > supported {
			return nil, fmt.Errorf("configured connector is not available, max available connectors %d", supported)
		}
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *OCPP) Enabled() (bool, error) {
	return c.cp.TransactionID() > 0, nil
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
	c.log.TRACE.Printf("SetChargingPriofileRequest %T: %+v", profile, profile)

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

var _ api.ChargePhases = (*Easee)(nil)

// Phases1p3p implements the api.ChargePhases interface
func (c *OCPP) Phases1p3p(phases int) error {
	err := c.setPeriod(c.current, phases)
	if err == nil {
		c.phases = phases
	}
	return err
}

// var _ api.Meter = (*OCPP)(nil)

// // CurrentPower implements the api.Meter interface
// func (c *OCPP) CurrentPower() (float64, error) {
// 	return 0, errors.New("not implemented")
// }

// var _ api.MeterEnergy = (*OCPP)(nil)

// // TotalEnergy implements the api.MeterMeterEnergy interface
// func (c *OCPP) TotalEnergy() (float64, error) {
// 	return 0, errors.New("not implemented")
// }

// var _ api.MeterCurrent = (*OCPP)(nil)

// // Currents implements the api.MeterCurrent interface
// func (c *OCPP) Currents() (float64, float64, float64, error) {
// 	return 0, 0, 0, errors.New("not implemented")
// }

// var _ api.Identifier = (*OCPP)(nil)

// // Identify implements the api.Identifier interface
// func (c *OCPP) Identify() (string, error) {
// 	return "", errors.New("not implemented")
// }
