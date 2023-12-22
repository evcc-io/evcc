package charger

import (
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// OpenWB20 charger implementation
type OpenWB20 struct {
	conn *modbus.Connection
	curr uint16
	base uint16
}

const (
	openwbRegPower        = 10100
	openwbRegImport       = 10102
	openwbRegVoltages     = 10104
	openwbRegCurrents     = 10107
	openwbRegPlugged      = 10114
	openwbRegCharging     = 10115
	openwbRegActualAmps   = 10116
	openwbRegSerial       = 10150
	openwbRegRfid         = 10160
	openwbRegCurrent      = 10171
	openwbRegPhaseTarget  = 10180
	openwbRegPhaseTrigger = 10181
	openwbRegHeartbeat    = 10190
	openwbRegCpTrigger    = 10198
)

func init() {
	registry.Add("openwb-2.0", NewOpenWB20FromConfig)
}

// https://openwb.de/main/wp-content/uploads/2023/10/ModbusTCP-openWB-series2-Pro-1.pdf

//go:generate go run ../cmd/tools/decorate.go -f decorateOpenWB20 -b *OpenWB20 -r api.Charger -t "api.Identifier,Identify,func() (string, error)"

// NewOpenWB20FromConfig creates a OpenWB20 charger from generic config
func NewOpenWB20FromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Connector          uint16
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		Connector: 1,
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewOpenWB20(cc.URI, cc.ID, cc.Connector)
	if err != nil {
		return nil, err
	}

	var identify func() (string, error)
	if _, err := wb.identify(); err == nil {
		identify = wb.identify
	}

	return decorateOpenWB20(wb, identify), nil
}

// NewOpenWB20 creates OpenWB20 charger
func NewOpenWB20(uri string, slaveID uint8, connector uint16) (*OpenWB20, error) {
	uri = util.DefaultPort(uri, 1502)

	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("openwb-2.0")
	conn.Logger(log.TRACE)

	wb := &OpenWB20{
		conn: conn,
		curr: 6 * 100,
		base: (connector - 1) * 100,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *OpenWB20) Status() (api.ChargeStatus, error) {
	if b, err := wb.conn.ReadInputRegisters(wb.base+openwbRegCharging, 1); err != nil || binary.BigEndian.Uint16(b) == 1 {
		return api.StatusC, err
	}

	if b, err := wb.conn.ReadInputRegisters(wb.base+openwbRegPlugged, 1); err != nil || binary.BigEndian.Uint16(b) == 1 {
		return api.StatusB, err
	}

	return api.StatusA, nil
}

// Enabled implements the api.Charger interface
func (wb *OpenWB20) Enabled() (bool, error) {
	b, err := wb.conn.ReadInputRegisters(wb.base+openwbRegActualAmps, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) == 1, nil
}

func (wb *OpenWB20) setCurrent(u uint16) error {
	_, err := wb.conn.WriteSingleRegister(wb.base+openwbRegCurrent, u)
	return err
}

// Enable implements the api.Charger interface
func (wb *OpenWB20) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.curr
	}

	return wb.setCurrent(u)
}

// MaxCurrent implements the api.Charger interface
func (wb *OpenWB20) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*OpenWB20)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *OpenWB20) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	curr := uint16(current * 100)
	err := wb.setCurrent(curr)
	if err == nil {
		wb.curr = curr
	}

	return err
}

var _ api.Meter = (*OpenWB20)(nil)

// CurrentPower implements the api.Meter interface
func (wb *OpenWB20) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wb.base+openwbRegPower, 2)
	if err != nil {
		return 0, err
	}
	return float64(int32(binary.BigEndian.Uint32(b))), nil
}

var _ api.MeterEnergy = (*OpenWB20)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *OpenWB20) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wb.base+openwbRegImport, 2)
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

// getPhaseValues returns phase values
func (wb *OpenWB20) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) / 100
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*OpenWB20)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *OpenWB20) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(wb.base + openwbRegCurrents)
}

var _ api.PhaseVoltages = (*OpenWB20)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *OpenWB20) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(wb.base + openwbRegVoltages)
}

var _ api.PhaseSwitcher = (*OpenWB20)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *OpenWB20) Phases1p3p(phases int) error {
	if _, err := wb.conn.WriteSingleRegister(wb.base+openwbRegPhaseTarget, uint16(phases)); err != nil {
		return err
	}

	_, err := wb.conn.WriteSingleRegister(wb.base+openwbRegPhaseTrigger, 1)
	return err
}

var _ api.Resurrector = (*OpenWB20)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *OpenWB20) WakeUp() error {
	_, err := wb.conn.WriteSingleRegister(wb.base+openwbRegCpTrigger, 1)
	return err
}

// Identify implements the api.Identifier interface
func (wb *OpenWB20) identify() (string, error) {
	b, err := wb.conn.ReadInputRegisters(wb.base+openwbRegRfid, 10)
	if err != nil {
		return "", err
	}
	return utf16BytesToString(b, binary.BigEndian), nil
}
