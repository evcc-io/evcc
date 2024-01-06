package tasks

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	gridx "github.com/grid-x/modbus"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

const Modbus TaskType = "modbus"

func init() {
	registry.Add(Modbus, ModbusHandlerFactory)
}

type ModbusResult struct {
	SlaveID uint8
	Model   int         `json:",omitempty"`
	Point   string      `json:",omitempty"`
	Value   interface{} `json:",omitempty"`
}

func (r *ModbusResult) Configuration(handler TaskHandler, res Result) map[string]interface{} {
	port := handler.(*ModbusHandler).Port
	cc := map[string]interface{}{
		"uri":   net.JoinHostPort(res.ResultDetails.IP, strconv.Itoa(port)),
		"model": "sunspec",
		"id":    r.SlaveID,
	}

	return cc
}

func ModbusHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := ModbusHandler{
		// Port:    502,
		IDs:     []uint8{1},
		Models:  []int{1},
		Point:   "Md", // Model
		Timeout: 10 * timeout,
	}

	err := util.DecodeOther(conf, &handler)

	if err == nil && len(handler.IDs) == 0 {
		err = errors.New("missing slave IDs")
	}

	if handler.Register.Address > 0 {
		handler.op, err = modbus.RegisterOperation(handler.Register)
	}

	return &handler, err
}

type ModbusHandler struct {
	Port     int
	IDs      []uint8
	Models   []int
	Point    string
	Register modbus.Register `mapstructure:",squash"`
	Values   []int
	Invalid  []int
	op       rs485.Operation
	Timeout  time.Duration
}

func (h *ModbusHandler) testRegister(_ *util.Logger, conn gridx.Client) bool {
	var bytes []byte
	var err error

	switch h.op.FuncCode {
	case gridx.FuncCodeReadHoldingRegisters:
		bytes, err = conn.ReadHoldingRegisters(h.op.OpCode, h.op.ReadLen)
	case gridx.FuncCodeReadInputRegisters:
		bytes, err = conn.ReadInputRegisters(h.op.OpCode, h.op.ReadLen)
	}

	if err != nil {
		return false
	}

	if len(h.Values) == 0 {
		return true
	}

	var u uint64
	switch h.op.ReadLen {
	case 1:
		u = uint64(binary.BigEndian.Uint16(bytes))
	case 2:
		u = uint64(binary.BigEndian.Uint32(bytes))
	case 4:
		u = binary.BigEndian.Uint64(bytes)
	}

	for _, val := range h.Values {
		if u == uint64(val) {
			return true
		}
	}

	return false
}

func (h *ModbusHandler) testSunSpec(log *util.Logger, conn meters.Connection, dev *sunspec.SunSpec, mr *ModbusResult) bool {
	err := dev.Initialize(conn.ModbusClient())
	if errors.Is(err, meters.ErrPartiallyOpened) {
		err = nil
	}
	if err != nil {
		return false
	}

	if len(h.Models) == 0 {
		return true
	}

	for _, model := range h.Models {
		_, res, err := dev.QueryPointAny(
			conn.ModbusClient(),
			model,
			0,
			h.Point,
		)

		if err == nil {
			mr.Model = model
			mr.Point = h.Point
			mr.Value = res.Value()

			log.DEBUG.Printf("model %d point %s: %v", model, mr.Point, mr.Value)

			if len(h.Invalid) == 0 {
				return true
			}

			var val int
			switch typ := res.Type(); typ {
			case "int16":
				val = int(res.Int16())
			case "uint16":
				val = int(res.Uint16())
			case "enum16":
				val = int(res.Enum16())
			case "count":
				val = int(res.Count())
			default:
				panic("invalid point type: " + typ)
			}

			for _, inv := range h.Invalid {
				if val != inv {
					return true
				}
			}
		} else {
			log.DEBUG.Printf("model %d: %v", model, err)
		}
	}

	return false
}

func (h *ModbusHandler) Test(log *util.Logger, in ResultDetails) (res []ResultDetails) {
	port := in.Port
	if port == 0 {
		port = h.Port
	}
	if port == 0 {
		fmt.Println("modbus", in)
		panic("modbus: invalid port")
	}

	addr := net.JoinHostPort(in.IP, strconv.Itoa(port))
	conn := meters.NewTCP(addr)
	dev := sunspec.NewDevice("sunspec")

	defer conn.Close()

	conn.Logger(log.TRACE)
	conn.Timeout(h.Timeout)

	for _, slaveID := range h.IDs {
		// grace period for id switch
		conn.Slave(slaveID)
		time.Sleep(100 * time.Millisecond)

		mr := ModbusResult{
			SlaveID: slaveID,
		}

		var ok bool
		if h.op.OpCode > 0 {
			// log.DEBUG.Printf("slave id: %d op: %v", slaveID, h.op)
			ok = h.testRegister(log, conn.ModbusClient())
		} else {
			// log.DEBUG.Printf("slave id: %d models: %v", slaveID, h.Models)
			ok = h.testSunSpec(log, conn, dev, &mr)
		}

		if ok {
			out := in.Clone()
			out.ModbusResult = &mr
			res = append(res, out)
		}
	}

	return res
}
