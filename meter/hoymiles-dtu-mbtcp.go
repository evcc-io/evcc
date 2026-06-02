package meter

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

const (
	hoymilesPanelLengthBytes   = uint16(0x28)
	hoymilesStartRegisterBytes = uint16(0x1000)
	hoymilesMaxPanels          = 99
)

func init() {
	registry.AddCtx("hoymiles-dtu-mb", NewHoymilesDTUModbusTcpFromConfig)
}

type hoymilesDTUValues struct {
	power       float64
	totalEnergy float64
}

type hoymilesDTUModbusTCP struct {
	log     *util.Logger
	conn    *modbus.Connection
	valuesG func() (hoymilesDTUValues, error)
}

// NewHoymilesDTUModbusTcpFromConfig creates a Hoymiles DTU meter from generic config
func NewHoymilesDTUModbusTcpFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		pvMaxACPower       `mapstructure:",squash"`
		Cache              time.Duration
	}{
		TcpSettings: modbus.TcpSettings{ID: 1},
		Cache:       15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return newHoymilesDTUModbusTCP(ctx, cc.URI, cc.ID, cc.pvMaxACPower.Decorator(), cc.Cache)
}

func newHoymilesDTUModbusTCP(ctx context.Context, uri string, id uint8, maxACPower func() float64, cache time.Duration) (api.Meter, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("hoymiles-dtu-mb")
	conn.Logger(log.TRACE)

	m := &hoymilesDTUModbusTCP{
		log:  log,
		conn: conn,
	}

	m.valuesG = util.Cached(m.readCurrentValues, cache)

	meter, _ := NewConfigurable(m.CurrentPower)
	implement.Has(meter, implement.MeterEnergy(m.TotalEnergy))
	implement.May(meter, implement.MaxACPowerGetter(maxACPower))

	return meter, nil
}

func (m *hoymilesDTUModbusTCP) panelPower(panelIndex int) (float64, float64, bool, error) {
	// Each panel's data is 0x28 bytes (40 bytes); panel N starts at register 0x1000 + N×0x28.
	// Confirmed with Hoymiles DTU PRO S hardware against the Modbus register map.
	startRegister := hoymilesStartRegisterBytes + uint16(panelIndex)*hoymilesPanelLengthBytes
	results, err := m.conn.ReadHoldingRegisters(startRegister, hoymilesPanelLengthBytes/2)
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to read panel %d: %w", panelIndex, err)
	}
	if len(results) < int(hoymilesPanelLengthBytes) {
		return 0, 0, false, fmt.Errorf("panel %d: short Modbus response: got %d bytes, expected %d", panelIndex, len(results), hoymilesPanelLengthBytes)
	}

	// inverterSerial: bytes 0x01–0x06
	serialBytes := results[1:7]
	inverterSerial := fmt.Sprintf("%X", serialBytes)
	// all-zero serial means no more panels; DTU returns zeros for unoccupied slots
	serialIsZero := binary.BigEndian.Uint32(serialBytes[:4]) == 0 &&
		binary.BigEndian.Uint16(serialBytes[4:6]) == 0
	// portNumber: byte 0x07
	portNumber := int(results[7])
	if serialIsZero || portNumber == 0 {
		m.log.TRACE.Printf("panel %d: no more panels to read", panelIndex)
		return 0, 0, false, nil
	}
	// power: bytes 0x10–0x11, uint16 / 10 → watts (one decimal place precision)
	power := float64(binary.BigEndian.Uint16(results[16:18])) / 10.0
	// totalCumulativeProduction: bytes 0x14–0x17, uint32, in Wh
	totalCumulativeProduction := binary.BigEndian.Uint32(results[20:24])
	m.log.TRACE.Printf("panel %d: inverter serial %s, port %d, power %.1f W, cumulative production %d Wh", panelIndex, inverterSerial, portNumber, power, totalCumulativeProduction)
	return power, float64(totalCumulativeProduction), true, nil
}

func (m *hoymilesDTUModbusTCP) readCurrentValues() (hoymilesDTUValues, error) {
	var values hoymilesDTUValues
	for i := 0; i < hoymilesMaxPanels; i++ {
		power, cumulative, found, err := m.panelPower(i)
		if err != nil {
			return hoymilesDTUValues{}, err
		}
		if !found {
			break
		}
		values.power += power
		values.totalEnergy += cumulative
	}
	m.log.DEBUG.Printf("total power from all panels: %.2f W, total cumulative production: %.2f Wh", values.power, values.totalEnergy)
	return values, nil
}

// CurrentPower implements the api.Meter interface
func (m *hoymilesDTUModbusTCP) CurrentPower() (float64, error) {
	res, err := m.valuesG()
	return res.power, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (m *hoymilesDTUModbusTCP) TotalEnergy() (float64, error) {
	res, err := m.valuesG()
	return res.totalEnergy / 1e3, err // Wh to kWh
}
