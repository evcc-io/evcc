package charger

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// OCPP charger implementation
type OCPP struct {
	log     *util.Logger
	cp      *ocpp.CP
	id      string
	enabled bool // TODO remove
}

func init() {
	registry.Add("ocpp", NewOCPPFromConfig)
}

// NewOCPPFromConfig creates a OCPP charger from generic config
func NewOCPPFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		StationId string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOCPP(cc.StationId)
}

// NewOCPP creates OCPP charger
func NewOCPP(id string) (*OCPP, error) {
	c := &OCPP{
		log: util.NewLogger("ocpp-" + id),
		cp:  ocpp.Instance().Register(id),
		id:  id,
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *OCPP) Enabled() (bool, error) {
	// TODO implement
	return c.enabled, nil
}

// Enable implements the api.Charger interface
func (c *OCPP) Enable(enable bool) error {
	c.enabled = enable
	if !enable {
		return nil
	}

	rc := make(chan error, 1)
	err := ocpp.Instance().CS().RemoteStartTransaction(c.id, func(request *core.RemoteStartTransactionConfirmation, err error) {
		c.log.TRACE.Printf("RemoteStartTransaction %T: %+v", request, request)

		if err == nil && request.Status != types.RemoteStartStopStatusAccepted {
			err = fmt.Errorf("invalid status: %s", request.Status)
		}

		rc <- err
		close(rc)
	}, "idTag")

	if err == nil {
		c.log.TRACE.Println("RemoteStartTransaction: waiting for response")
		err = <-rc
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *OCPP) MaxCurrent(current int64) error {
	// TODO implement
	return nil
}

// Status implements the api.Charger interface
func (c *OCPP) Status() (api.ChargeStatus, error) {
	return api.StatusB, errors.New("not implemented")
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
