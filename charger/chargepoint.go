package charger

import (
	"fmt"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	cpkg "github.com/evcc-io/evcc/charger/chargepoint"
	"github.com/evcc-io/evcc/util"
)

var _ api.Charger = (*ChargePoint)(nil)

func init() {
	registry.Add("chargepoint", NewChargePointFromConfig)
}

// ChargePoint implements the api.Charger interface for ChargePoint Home Flex chargers.
type ChargePoint struct {
	*cpkg.API
	deviceID int
	enabled  bool
	statusG  util.Cacheable[cpkg.HomeChargerStatus]
}

// NewChargePointFromConfig creates a ChargePoint charger from generic config.
func NewChargePointFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		DeviceID   int
		User       string
		Password   string
		MinCurrent int64
		MaxCurrent int64
		Cache      time.Duration
	}{
		MinCurrent: 8,
		MaxCurrent: 48,
		Cache:      30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewChargePoint(cc.DeviceID, cc.User, cc.Password, cc.MinCurrent, cc.MaxCurrent, cc.Cache)
}

// NewChargePoint creates a ChargePoint Home Flex charger.
func NewChargePoint(deviceID int, user, password string, minCurrent, maxCurrent int64, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("chargepoint").Redact(user, password)

	identity, err := cpkg.NewIdentity(log, user, password)
	if err != nil {
		return nil, fmt.Errorf("identity: %w", err)
	}

	api := cpkg.NewAPI(log, identity)

	id := ""
	if deviceID != 0 {
		id = strconv.Itoa(deviceID)
	}
	foundID, err := ensureChargerEx(id, api.HomeChargerIDs, func(v int) (string, error) {
		return strconv.Itoa(v), nil
	})
	if err != nil {
		return nil, fmt.Errorf("charger: %w", err)
	}
	deviceID = foundID

	cp := &ChargePoint{
		API:      api,
		deviceID: deviceID,
	}

	cp.statusG = util.ResettableCached(func() (cpkg.HomeChargerStatus, error) {
		return cp.API.HomeChargerStatus(cp.deviceID)
	}, cache)

	return cp, nil
}

// Status implements the api.Charger interface.
func (c *ChargePoint) Status() (api.ChargeStatus, error) {
	res, err := c.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	switch {
	case res.ChargingStatus == "CHARGING":
		return api.StatusC, nil
	case res.IsPluggedIn:
		return api.StatusB, nil // Connected
	default:
		return api.StatusA, nil // Disconnected
	}
}

// Enabled implements the api.Charger interface.
func (c *ChargePoint) Enabled() (bool, error) {
	return verifyEnabled(c, c.enabled)
}

// Enable implements the api.Charger interface.
func (c *ChargePoint) Enable(enable bool) error {
	api := c.API.StopCharging
	if enable {
		api = c.API.StartCharging
	}
	err := api(c.deviceID)
	if err != nil {
		return err
	}

	c.enabled = enable
	c.statusG.Reset()
	return nil
}

// MaxCurrent implements the api.Charger interface.
func (c *ChargePoint) MaxCurrent(current int64) error {
	if err := c.API.SetAmperageLimit(c.deviceID, current); err != nil {
		return err
	}

	c.statusG.Reset()
	return nil
}
