package charger

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util"
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

	err := cp.Boot()

	return c, err
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
		err = ocpp.Instance().CS().RemoteStartTransaction(c.id, func(resp *core.RemoteStartTransactionConfirmation, err error) {
			c.log.TRACE.Printf("RemoteStartTransaction %T: %+v", resp, resp)

			if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
				err = fmt.Errorf("invalid status: %s", resp.Status)
			}

			rc <- err
			close(rc)
		}, c.idtag, func(request *core.RemoteStartTransactionRequest) {
			request.ConnectorId = &c.connector
		})
	} else {
		err = ocpp.Instance().CS().RemoteStopTransaction(c.id, func(resp *core.RemoteStopTransactionConfirmation, err error) {
			c.log.TRACE.Printf("RemoteStopTransaction %T: %+v", resp, resp)

			if err == nil && resp != nil && resp.Status != types.RemoteStartStopStatusAccepted {
				err = fmt.Errorf("invalid status: %s", resp.Status)
			}

			rc <- err
			close(rc)
		}, c.cp.TransactionID())
	}

	if err == nil {
		c.log.TRACE.Println("RemoteStartStopTransaction: waiting for response")
		err = <-rc
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *OCPP) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

// setPeriod sets a single charging schedule period with given current and phases
func (c *OCPP) setPeriod(current float64, phases int) error {
	if current == 0 {
		current = c.current
	}

	period := types.ChargingSchedulePeriod{
		StartPeriod: 1,
		Limit:       current,
	}

	if phases == 0 {
		phases = c.phases
	}

	if phases > 0 {
		period.NumberPhases = &phases
	}

	rc := make(chan error, 1)
	err := ocpp.Instance().CS().SetChargingProfile(c.id, func(resp *smartcharging.SetChargingProfileConfirmation, err error) {
		c.log.TRACE.Printf("SetChargingProfile %T: %+v", resp, resp)

		if err == nil && resp != nil && resp.Status != smartcharging.ChargingProfileStatusAccepted {
			err = fmt.Errorf("invalid status: %s", resp.Status)
		}

		rc <- err
		close(rc)
	}, c.connector, &types.ChargingProfile{
		ChargingProfileId:      1,
		StackLevel:             1,
		ChargingProfilePurpose: types.ChargingProfilePurposeChargePointMaxProfile,
		ChargingProfileKind:    types.ChargingProfileKindAbsolute,
		ChargingSchedule: &types.ChargingSchedule{
			ChargingRateUnit:       types.ChargingRateUnitAmperes,
			ChargingSchedulePeriod: []types.ChargingSchedulePeriod{period},
		},
	})

	if err == nil {
		c.log.TRACE.Println("SetChargingProfile: waiting for response")
		err = <-rc
	}

	return err
}

// MaxCurrent implements the api.ChargerEx interface
func (c *OCPP) MaxCurrentMillis(current float64) error {
	err := c.setPeriod(current, 0)
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
	err := c.setPeriod(0, phases)
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
