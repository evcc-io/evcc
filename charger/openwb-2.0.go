package charger

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openwb/pro"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/request"
)

// OpenWB20 charger implementation
type OpenWB20 struct {
	conn       *modbus.Connection
	enabled    bool
	curr       uint16
	base       uint16
	httpHelper *request.Helper
	httpUri    string
	statusG    util.Cacheable[pro.Status]
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
	registry.AddCtx("openwb-2.0", NewOpenWB20FromConfig)
}

// https://openwb.de/main/wp-content/uploads/2023/10/ModbusTCP-openWB-series2-Pro-1.pdf

//go:generate go tool decorate -f decorateOpenWB20 -b *OpenWB20 -r api.Charger -t api.PhaseSwitcher,api.Identifier,api.Battery

// NewOpenWB20FromConfig creates a OpenWB20 charger from generic config
func NewOpenWB20FromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		Connector          uint16
		Phases1p3p         bool
		HttpFallback       bool
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

	wb, err := NewOpenWB20(ctx, cc.URI, cc.ID, cc.Connector, cc.HttpFallback)
	if err != nil {
		return nil, err
	}

	var phases1p3p func(int) error
	if cc.Phases1p3p {
		phases1p3p = wb.phases1p3p
	}

	var identify func() (string, error)
	if _, err := wb.identify(); err == nil {
		identify = wb.identify
	}

	var soc func() (float64, error)
	if wb.httpHelper != nil {
		soc = wb.soc
	}

	return decorateOpenWB20(wb, phases1p3p, identify, soc), nil
}

// NewOpenWB20 creates OpenWB20 charger
func NewOpenWB20(ctx context.Context, uri string, slaveID uint8, connector uint16, httpFallback bool) (*OpenWB20, error) {
	uri = util.DefaultPort(uri, 1502)

	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
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

	if httpFallback {
		host, _, err := net.SplitHostPort(uri)
		if err != nil {
			return nil, fmt.Errorf("http fallback: invalid modbus uri: %w", err)
		}
		httpUri := fmt.Sprintf("http://%s:8080", host)
		wb.httpHelper = request.NewHelper(log)
		wb.httpUri = strings.TrimRight(httpUri, "/")
		wb.statusG = util.ResettableCached(func() (pro.Status, error) {
			var res pro.Status
			url := fmt.Sprintf("%s/%s", wb.httpUri, "connect.php")
			err := wb.httpHelper.GetJSON(url, &res)
			return res, err
		}, time.Second)
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
	return verifyEnabled(wb, wb.enabled)
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
	err := wb.setCurrent(u)
	if err == nil {
		wb.enabled = enable
	}
	return err
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
		res[i] = float64(int16(binary.BigEndian.Uint16(b[2*i:]))) / 100
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

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *OpenWB20) phases1p3p(phases int) error {
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

// getHttpStatus returns status from connect.php (openWB-Pro+ only)
func (wb *OpenWB20) getHttpStatus() (pro.Status, error) {
	if wb.statusG == nil {
		return pro.Status{}, fmt.Errorf("http fallback not configured")
	}
	return wb.statusG.Get()
}

// identify implements the api.Identifier interface
func (wb *OpenWB20) identify() (string, error) {
	b, modbusErr := wb.conn.ReadInputRegisters(wb.base+openwbRegRfid, 10)
	if modbusErr == nil {
		if id := bytesAsString(b); id != "" {
			return id, nil
		}
	}

	if wb.httpHelper != nil {
		res, httpErr := wb.getHttpStatus()
		if httpErr == nil {
			if res.VehicleID != "" && res.VehicleID != "--" {
				return res.VehicleID, nil
			}
			if res.RfidTag != "" {
				return res.RfidTag, nil
			}
		}
		if httpErr != nil {
			return "", httpErr
		}
	}

	if modbusErr != nil {
		return "", modbusErr
	}
	return "", nil
}

// soc implements the api.Battery interface (openWB-Pro+ only)
func (wb *OpenWB20) soc() (float64, error) {
	if wb.httpHelper == nil {
		return 0, api.ErrNotAvailable
	}

	res, err := wb.getHttpStatus()
	if err != nil {
		return 0, err
	}

	if time.Since(time.Unix(res.SocTimestamp, 0)) > 5*time.Minute {
		return 0, api.ErrNotAvailable
	}

	return float64(res.Soc), nil
}
