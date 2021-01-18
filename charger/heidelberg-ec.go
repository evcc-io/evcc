package charger

import (
	"errors"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

// HeidelbergEC charger implementation
type HeidelbergEC struct {
	log     *util.Logger
	conn    *modbus.Connection
	current int64
}

const (
	wbRegAmpsConfig    = 40257
	wbRegVehicleStatus = 30006
)

func init() {
	registry.Add("HeidelbergEC", NewHeidelbergECFromConfig)
}

// https://cdn.shopify.com/s/files/1/0101/2409/9669/files/heidelberg-energy-control-modbus.pdf

// NewHeidelbergECFromConfig creates a HeidelbergEC charger from generic config
func NewHeidelbergECFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		Baudrate: 19200,
		Comset:   "8E1",
		ID:       1,
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHeidelbergEC(cc.URI, cc.Device, cc.Comset, cc.Baudrate, true, cc.ID)
}

// NewHeidelbergEC creates HeidelbergEC charger
func NewHeidelbergEC(uri, device, comset string, baudrate int, rtu bool, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, rtu, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("heidelberg")
	conn.Logger(log.TRACE)

	wb := &HeidelbergEC{
		log:     log,
		conn:    conn,
		current: 6, // assume min current
	}

	return wb, nil
}

// Status implements the Charger.Status interface
func (wb *HeidelbergEC) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegVehicleStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch b {
	case 1: // undefined
		return api.StatusNone, nil
	case 2: // A1 standby
		return api.StatusA, nil
	case 3: // A2 standby
		return api.StatusA, nil
	case 4: // B1 vehicle detected
		return api.StatusB, nil
	case 5: // B2 vehicle detected
		return api.StatusB, nil
	case 6: // C1 ready (charging)
		return api.StatusC, nil
	case 7: // C2 ready (charging)
		return api.StatusC, nil
	case 8: // D charging with ventilation
		return api.StatusD, nil
	case 9: // E no power (shut off)
		return api.StatusE, nil
	case 10: // F error
		return api.StatusE, nil
	case 11: // ERR error
		return api.StatusE, nil
	default:
		return api.StatusNone, errors.New("invalid response")
	}
}

// Enabled implements the Charger.Enabled interface
func (wb *HeidelbergEC) Enabled() (bool, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegAmpsConfig, 1)
	if err != nil {
		return false, err
	}

	enabled := b != 0
	if enabled {
		wb.current = int64(b)
	}

	return enabled, nil
}

// Enable implements the Charger.Enable interface
func (wb *HeidelbergEC) Enable(enable bool) error {
	b := 0

	if enable {
		b = wb.current
	}

	_, err := wb.conn.WriteSingleRegister(wbRegAmpsConfig, uint16(b))

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *HeidelbergEC) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(wbRegAmpsConfig, uint16(current))
	if err == nil {
		wb.current = current
	}

	return err
}
