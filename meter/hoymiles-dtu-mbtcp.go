package meter

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

func init() {
	registry.Add("hoymiles-dtu-mb", NewHoymilesDTUModbusTcpFromConfig)
}

type HoymilesDTUModbusTcpConfig struct {
	Host       string
	Port       string
	ID         uint8
	MaxACPower int
	Cache      time.Duration
}

type hoymilesDTUValues struct {
	power       float64
	totalEnergy float64
}

type HoymilesDTUModbusTcp struct {
	log     *util.Logger
	conn    *modbus.Connection
	maxAC   float64
	valuesG func() (hoymilesDTUValues, error)
}

// NewHoymilesDTUModbusTcpFromConfig creates a Hoymiles DTU meter from generic config
func NewHoymilesDTUModbusTcpFromConfig(other map[string]any) (api.Meter, error) {
	cc := HoymilesDTUModbusTcpConfig{
		Port:  "502",
		ID:    1,
		Cache: time.Second * 15,
		// cache results for 15 seconds to avoid hitting the DTU too often
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	uri := net.JoinHostPort(cc.Host, cc.Port)

	modbus.Lock()
	defer modbus.Unlock()

	conn, err := modbus.NewConnection(context.Background(), uri, "", "", 0, modbus.Tcp, cc.ID)
	if err != nil {
		return nil, err
	}

	res := &HoymilesDTUModbusTcp{
		log:   util.NewLogger("hoymiles-dtu-mb"),
		conn:  conn,
		maxAC: float64(cc.MaxACPower),
	}

	res.valuesG = util.Cached(res.readCurrentValues, cc.Cache)

	maxACPower := pvMaxACPower{MaxACPower: float64(cc.MaxACPower)}
	return decorateMeter(res, res.TotalEnergy, nil, nil, nil, maxACPower.Decorator()), nil
}

var _ api.Meter = (*HoymilesDTUModbusTcp)(nil)

// Function to read single panel power from the DTU
// param panelIndex: the index of the panel to read (starting from 0),
// and modbus session, and returns the power in watts, or an error if the read fails
// returns the power in watts, totalCumulativeProduction in Wh, a flag indicating whether a panel exists, or an error if the read fails
func (m *HoymilesDTUModbusTcp) PanelPower(panelIndex int) (float64, float64, bool, error) {
	PORT_LENGTH := uint16(0x28)      // length of the data for each panel in bytes
	START_REGISTER := uint16(0x1000) // starting register for the first panel
	// calculate the starting register for the panel based on the index
	startRegister := START_REGISTER + uint16(panelIndex)*PORT_LENGTH
	// read PORT_LENGTH bytes from the DTU starting at startRegister
	results, err := m.conn.ReadHoldingRegisters(startRegister, PORT_LENGTH)
	if err != nil {
		return 0, 0, false, fmt.Errorf("Failed to read hoymiles-dtu panel %d: %w", panelIndex, err)
	}
	// check that results has at least 24 bytes (to read power and total cumulative production)
	if len(results) < 24 {
		return 0, 0, false, fmt.Errorf("Invalid response length for panel %d: expected at least 24 bytes, got %d", panelIndex, len(results))
	}

	// inverterSerial in 0x1 – 0x6, hex to string
	serialBytes := results[1:6]
	inverterSerial := fmt.Sprintf("%X", serialBytes)
	serialIsZero := binary.BigEndian.Uint32(serialBytes[:4]) == 0 && serialBytes[4] == 0
	// portNumber in 0x7, hex to int
	portNumber := int(results[7])
	if serialIsZero || portNumber == 0 {
		m.log.TRACE.Printf("Panel %d: No more panels to read", panelIndex)
		// if the serial number is all zeros or the port number is zero, we assume there are no more panels to read
		return 0, 0, false, nil
	}
	// power in 0x10 – 0x11, hex to uint16, divided by 10 to get actual power in watts (one decimal place precision)
	power := float64(binary.BigEndian.Uint16(results[16:18])) / 10.0
	// totalCumulativeProduction in bytes 0x14 – 0x17, hex to uint32, in Wh
	totalCumulativeProduction := binary.BigEndian.Uint32(results[20:24]) // in Wh
	m.log.TRACE.Printf("Panel %d: Inverter Serial: %s, Port Number: %d, Power: %.1f W, Total Cumulative Production: %d Wh", panelIndex, inverterSerial, portNumber, power, totalCumulativeProduction)
	return power, float64(totalCumulativeProduction), true, nil
}

func (m *HoymilesDTUModbusTcp) readCurrentValues() (hoymilesDTUValues, error) {
	var values hoymilesDTUValues
	for i := 0; i < 99; i++ {
		power, cumulative, found, err := m.PanelPower(i)
		if err != nil {
			return hoymilesDTUValues{}, fmt.Errorf("Failed to read hoymiles-dtu-panel %d: %w", i, err)
		}
		if !found {
			break
		}
		values.power += power
		values.totalEnergy += cumulative
	}
	m.log.DEBUG.Printf("Total power from all panels: %.2f W / %.2f W, Total cumulative production: %.2f Wh", values.power, m.maxAC, values.totalEnergy)
	return values, nil
}

// CurrentPower implements the api.Meter interface
func (m *HoymilesDTUModbusTcp) CurrentPower() (float64, error) {
	res, err := m.valuesG()
	return res.power, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (m *HoymilesDTUModbusTcp) TotalEnergy() (float64, error) {
	res, err := m.valuesG()
	return res.totalEnergy / 1e3, err // Wh to kWh
}
