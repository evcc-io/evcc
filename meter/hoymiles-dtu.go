package meter

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

func init() {
	registry.Add("hoymiles-dtu", NewHoymilesDTUFromConfig)
}

// 2. Define the configuration struct
// This automatically maps to the YAML settings
type HoymilesDTUConfig struct {
	Host string
	Port string
	ID   uint8
}

// 3. Define the main struct that holds your state
type HoymilesDTU struct {
	conn      *modbus.Connection
	registers []uint16 // We will store the discovered panel registers here
}

// NewHoymilesDTUFromConfig creates a Hoymiles DTU meter from generic config
// what we need to know:
// - IP address and port of the DTU (default 502)
func NewHoymilesDTUFromConfig(other map[string]any) (api.Meter, error) {
	cc := HoymilesDTUConfig{
		Port: "502",
		ID:   1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	uri := net.JoinHostPort(cc.Host, cc.Port)
	conn, err := modbus.NewConnection(context.Background(), uri, "", "", 0, modbus.Tcp, cc.ID)
	if err != nil {
		return nil, err
	}

	res := &HoymilesDTU{
		conn: conn,
	}

	return res, nil
}

// define noMorePanels type to indicate that there are no more panels to read
type noMorePanels struct{}

func (e noMorePanels) Error() string {
	return "no more panels"
}

// Function to read single panel power from the DTU
// param panelIndex: the index of the panel to read (starting from 0),
// and modbus session, and returns the power in watts, or an error if the read fails
// returns the power in watts, totalCumulativeProduction in Wh, or an error if the read fails
// returns noMorePanels if panelIndex is out of range
func (m *HoymilesDTU) PanelPower(panelIndex int, conn *modbus.Connection) (float64, float64, error) {
	PORT_LENGTH := uint16(0x28)      // length of the data for each panel in bytes
	START_REGISTER := uint16(0x1000) // starting register for the first panel
	// calculate the starting register for the panel based on the index
	startRegister := START_REGISTER + uint16(panelIndex)*PORT_LENGTH
	// read PORT_LENGTH bytes from the DTU starting at startRegister
	results, err := conn.ReadHoldingRegisters(startRegister, PORT_LENGTH)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read panel %d: %w", panelIndex, err)
	}
	// inverterSerial in 0x1 - 0x6, hex to string
	inverterSerial := string(results[1:6])
	// portNumber in 0x7, hex to int
	portNumber := int(results[7])
	if inverterSerial == "" || portNumber == 0 {
		return 0, 0, noMorePanels{}
	}
	// power in 0x10 - 0x11, hex to int, divided by 10 to get actual power in watts
	power := binary.BigEndian.Uint16(results[16:18]) / 10 // power is located at bytes 16-17 of the panel data, and is diveded by 10 to get the actual power in watts
	// totalCumulativeProduction in 20 - 24
	totalCumulativeProduction := binary.BigEndian.Uint32(results[20:24]) // total cumulative production is located at bytes 20-23 of the panel data, and is in Wh
	return float64(power), float64(totalCumulativeProduction), nil
}

func (m *HoymilesDTU) ReadCurrentValues() (float64, float64, error) {
	// panel power is the sum of all panels, so we need to read each panel and sum their power
	totalPower := 0.0
	totalCumulativeProduction := 0.0
	for i := 0; i < 99; i++ { // we will read up to 99 panels, but will stop if we get no more panels error
		power, cumulative, err := m.PanelPower(i, m.conn)
		if err != nil {
			if _, ok := err.(noMorePanels); ok {
				break // no more panels to read, exit the loop
			}
			return 0, 0, fmt.Errorf("failed to read panel %d: %w", i, err)
		}
		totalPower += power
		totalCumulativeProduction += cumulative
	}

	return totalPower, totalCumulativeProduction, nil
}

func (m *HoymilesDTU) CurrentPower() (float64, error) {
	power, _, err := m.ReadCurrentValues()
	return power, err
}

func (m *HoymilesDTU) TotalCumulativeProduction() (float64, error) {
	_, cumulative, err := m.ReadCurrentValues()
	return cumulative, err
}
