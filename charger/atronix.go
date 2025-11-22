package charger

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type Atronix struct {
	*OCPP
	pendingStart bool
}

var (
	_ api.ChargerEx     = (*Atronix)(nil)
	_ api.Meter         = (*Atronix)(nil)
	_ api.MeterEnergy   = (*Atronix)(nil)
	_ api.PhaseCurrents = (*Atronix)(nil)
	_ api.PhaseVoltages = (*Atronix)(nil)
	_ api.CurrentGetter = (*Atronix)(nil)
	_ api.Battery       = (*Atronix)(nil)
)

func init() {
	registry.AddCtx("atronix-ocpp", NewAtronixFromConfig)
}

func NewAtronixFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		StationId      string
		IdTag          string
		Connector      int
		MeterInterval  time.Duration
		MeterValues    string
		ConnectTimeout time.Duration

		Timeout          time.Duration
		BootNotification *bool
		GetConfiguration *bool
		ChargingRateUnit types.ChargingRateUnitType
		AutoStart        bool
		NoStop           bool

		ForcePowerCtrl      bool
		StackLevelZero      *bool
		ProfileKindRelative bool
		RemoteStart         bool
	}{
		Connector:      1,
		MeterInterval:  10 * time.Second,
		ConnectTimeout: 5 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	stackLevelZero := cc.StackLevelZero != nil && *cc.StackLevelZero
	profileKindRelative := cc.ProfileKindRelative

	base, err := NewOCPP(ctx,
		cc.StationId, cc.Connector, cc.IdTag,
		cc.MeterValues, cc.MeterInterval,
		cc.ForcePowerCtrl, stackLevelZero, profileKindRelative, cc.RemoteStart,
		cc.ConnectTimeout,
	)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	return &Atronix{OCPP: base}, nil
}

func (c *Atronix) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

func (c *Atronix) Enabled() (bool, error) {
	return c.OCPP.Enabled()
}

func (c *Atronix) Enable(enable bool) error {
	if enable {
		if err := c.OCPP.Enable(true); err != nil {
			return err
		}

		if idTag := c.conn.IdTag(); idTag != "" {
			if txn, err := c.conn.TransactionID(); err == nil && txn == 0 {
				if err := c.conn.RemoteStartTransactionRequest(idTag); err == nil {
					c.pendingStart = true
				} else {
					c.log.WARN.Printf("ocpp/atronix: remote start on enable failed: %v", err)
				}
			}
		}

		return nil
	}

	txn, err := c.conn.TransactionID()
	if err != nil {
		return fmt.Errorf("ocpp/atronix: transaction lookup failed: %w", err)
	}
	if txn > 0 {
		if err := c.conn.RemoteStopTransactionRequest(txn); err != nil {
			return fmt.Errorf("ocpp/atronix: remote stop failed: %w", err)
		}
	}

	c.pendingStart = false
	c.current = 0

	_ = c.setCurrent(0)

	return c.OCPP.Enable(false)
}

func (c *Atronix) MaxCurrentMillis(current float64) error {
	vendor, model := c.vendorModel()

	if vendor != "" || model != "" {
		if strings.Contains(vendor, "weeyu") || strings.Contains(model, "atronix") {
			if current <= 0 {
				txn, err := c.conn.TransactionID()
				if err != nil {
					return fmt.Errorf("ocpp/atronix: transaction lookup failed: %w", err)
				}

				if txn > 0 {
					if err := c.conn.RemoteStopTransactionRequest(txn); err != nil {
						return fmt.Errorf("ocpp/atronix: remote stop failed: %w", err)
					}
				}

				c.pendingStart = false
				c.current = 0
				c.enabled = false
				return nil
			}

			txn, err := c.conn.TransactionID()
			if err != nil {
				return fmt.Errorf("ocpp/atronix: transaction lookup failed: %w", err)
			}

			if txn == 0 {
				if !c.pendingStart {
					if idTag := c.conn.IdTag(); idTag != "" {
						if err := c.conn.RemoteStartTransactionRequest(idTag); err != nil {
							return fmt.Errorf("ocpp/atronix: remote start failed: %w", err)
						}
						c.pendingStart = true
					}
				}
			} else {
				c.pendingStart = false
			}

			val := strconv.FormatFloat(current, 'f', 0, 64)

			if err := c.cp.ChangeConfigurationRequest("ChargeRate", val); err == nil {
				c.log.INFO.Printf("ocpp/atronix: ChargeRate=%s A (accepted)", val)
				c.current = current
				return nil
			} else if strings.Contains(strings.ToLower(err.Error()), "notsupported") {
				c.log.WARN.Printf("ocpp/atronix: ChargeRate not supported, fallback to charging profile (%v)", err)
			} else {
				return fmt.Errorf("ocpp/atronix: ChangeConfiguration ChargeRate=%s failed: %w", val, err)
			}
		}
	}

	return c.OCPP.MaxCurrentMillis(current)
}

func (c *Atronix) CurrentPower() (float64, error) {
	return c.conn.CurrentPower()
}

func (c *Atronix) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}

func (c *Atronix) Currents() (float64, float64, float64, error) {
	return c.conn.Currents()
}

func (c *Atronix) Voltages() (float64, float64, float64, error) {
	return c.conn.Voltages()
}

func (c *Atronix) GetMaxCurrent() (float64, error) {
	return c.conn.GetMaxCurrent()
}

func (c *Atronix) Soc() (float64, error) {
	return c.conn.Soc()
}

func (c *Atronix) vendorModel() (string, string) {
	if c.cp == nil || c.cp.BootNotificationResult == nil {
		return "", ""
	}

	return strings.ToLower(c.cp.BootNotificationResult.ChargePointVendor),
		strings.ToLower(c.cp.BootNotificationResult.ChargePointModel)
}
