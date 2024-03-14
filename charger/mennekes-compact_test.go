package charger_test

import (
	"encoding/binary"
	"io"
	"testing"
	"time"

	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus/test"
)

func TestMennekesCompact_Phases1p3p(t *testing.T) {
	mennekesInt := func(u uint16) []byte { // mennekes byte order is big endian
		return binary.BigEndian.AppendUint16(nil, u)
	}

	var phases uint16 = 1

	modbusClient := &test.ModbusTestClient{
		OnReadFn: func(addr, quantity uint16) (results []byte, err error) {
			t.Logf("read %v %v", addr, quantity)
			switch addr {
			case 0x0314: // phase pause duration
				return []byte{0x00, 0x00}, nil
			case 0x0D03: // chargingMode
				if phases == 1 {
					return mennekesInt(1), nil
				}
				return mennekesInt(2), nil
			default:
				return []byte{0x00, 0x00}, nil
			}
		},
		OnWriteFn: func(addr uint16, value []byte) (results []byte, err error) {
			t.Logf("write %v %v", addr, value)
			switch addr {
			case 0x0D04: // phase switch reg
				if phases != binary.BigEndian.Uint16(value) {
					t.Errorf("unexpected phase switch value: %v", value)
				}
				return mennekesInt(phases), nil
			case 0x0D00: // heartbeat
				fallthrough
			default:
				return value, nil
			}
		},
	}

	log := util.NewLogger("test")
	log.SetLogOutput(io.Discard)
	apiCharger := charger.NewMennekesCompactWithModbusClient(&charger.MennekesCompactConfig{}, log, modbusClient)
	mc := apiCharger.(*charger.MennekesCompact)

	err := mc.Phases1p3p(1)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second)

}
