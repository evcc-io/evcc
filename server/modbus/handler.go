package modbus

import (
	"encoding/binary"
	"errors"
	"math/bits"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	gridx "github.com/grid-x/modbus"
)

type handler struct {
	log      *util.Logger
	readOnly bool
	mbserver.RequestHandler
	conn *modbus.Connection
}

func bytesAsUint16(b []byte) []uint16 {
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i < len(b)/2; i++ {
		u = append(u, binary.BigEndian.Uint16(b[2*i:]))
	}
	return u
}

func asBytes(u []uint16) []byte {
	b := make([]byte, 2*len(u))
	for i, u := range u {
		binary.BigEndian.PutUint16(b[2*i:], u)
	}
	return b
}

func (h *handler) logResult(op string, b []byte, err error) {
	if err == nil {
		h.log.TRACE.Printf(op+": %0x", b)
	} else {
		h.log.TRACE.Printf(op+": %v", err)
	}
}

func (h *handler) exceptionToUint16AndError(op string, b []byte, err error) ([]uint16, error) {
	h.logResult(op, b, err)

	var modbusError *gridx.Error
	if errors.As(err, &modbusError) {
		err = mbserver.MapExceptionCodeToError(modbusError.ExceptionCode)
	}

	return bytesAsUint16(b), err
}

func coilsToBytes(b []bool) []byte {
	l := len(b) / 8
	if len(b)%8 != 0 {
		l++
	}

	res := make([]byte, l)

	for i, bb := range b {
		if bb {
			byteNum := i / 8
			bit := i % 8

			res[byteNum] |= bits.RotateLeft8(1, bit)
		}
	}

	return res
}

func (h *handler) coilsToResult(op string, qty uint16, b []byte, err error) ([]bool, error) {
	h.logResult(op, b, err)

	var modbusError *gridx.Error
	if errors.As(err, &modbusError) {
		err = mbserver.MapExceptionCodeToError(modbusError.ExceptionCode)
	}

	var res []bool

LOOP:
	for _, bb := range b {
		for bit := 0; bit < 8; bit++ {
			if len(res) >= int(qty) {
				break LOOP
			}

			res = append(res, bits.RotateLeft8(bb, -bit)&1 != 0)
		}
	}

	return res, err
}

func (h *handler) HandleCoils(req *mbserver.CoilsRequest) ([]bool, error) {
	if req.IsWrite {
		if h.readOnly {
			return nil, mbserver.ErrIllegalFunction
		}

		if req.WriteFuncCode == gridx.FuncCodeWriteSingleCoil {
			h.log.TRACE.Printf("write single coil: id %d addr %d val %t", req.UnitId, req.Addr, req.Args[0])
			var u uint16
			if req.Args[0] {
				u = 0xFF00
			}

			b, err := h.conn.WriteSingleCoilWithSlave(req.UnitId, req.Addr, u)
			return h.coilsToResult("write coil", req.Quantity, b, err)
		}

		h.log.TRACE.Printf("write coils: id %d addr %d qty %d val %v", req.UnitId, req.Addr, req.Quantity, req.Args)
		args := coilsToBytes(req.Args)
		b, err := h.conn.WriteMultipleCoilsWithSlave(req.UnitId, req.Addr, req.Quantity, args)
		return h.coilsToResult("write coils", req.Quantity, b, err)
	}

	h.log.TRACE.Printf("read coil: id %d addr %d qty %d", req.UnitId, req.Addr, req.Quantity)
	b, err := h.conn.ReadCoilsWithSlave(req.UnitId, req.Addr, req.Quantity)
	return h.coilsToResult("read coil", req.Quantity, b, err)
}

func (h *handler) HandleInputRegisters(req *mbserver.InputRegistersRequest) (res []uint16, err error) {
	h.log.TRACE.Printf("read input: id %d addr %d qty %d", req.UnitId, req.Addr, req.Quantity)
	b, err := h.conn.ReadInputRegistersWithSlave(req.UnitId, req.Addr, req.Quantity)
	return h.exceptionToUint16AndError("read input", b, err)
}

func (h *handler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) (res []uint16, err error) {
	if req.IsWrite {
		if h.readOnly {
			return nil, mbserver.ErrIllegalFunction
		}

		if req.WriteFuncCode == gridx.FuncCodeWriteSingleRegister {
			h.log.TRACE.Printf("write holding: id %d addr %d val %04x", req.UnitId, req.Addr, req.Args[0])
			b, err := h.conn.WriteSingleRegisterWithSlave(req.UnitId, req.Addr, req.Args[0])
			return h.exceptionToUint16AndError("write holding", b, err)
		}

		h.log.TRACE.Printf("write holding: id %d addr %d qty %d val %0x", req.UnitId, req.Addr, req.Quantity, asBytes(req.Args))
		b, err := h.conn.WriteMultipleRegistersWithSlave(req.UnitId, req.Addr, req.Quantity, asBytes(req.Args))
		return h.exceptionToUint16AndError("write multiple holding", b, err)
	}

	h.log.TRACE.Printf("read holding: id %d addr %d qty %d", req.UnitId, req.Addr, req.Quantity)
	b, err := h.conn.ReadHoldingRegistersWithSlave(req.UnitId, req.Addr, req.Quantity)
	return h.exceptionToUint16AndError("read holding", b, err)
}
