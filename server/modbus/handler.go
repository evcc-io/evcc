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
	readOnly ReadOnlyMode
	conn     *modbus.Connection
}

func bytesAsUint16(b []byte) []uint16 {
	u := make([]uint16, 0, len(b)/2)
	for i := range len(b) / 2 {
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

func (h *handler) bytesToBoolResult(op string, qty uint16, b []byte, err error) ([]bool, error) {
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

func (h *handler) HandleDiscreteInputs(req *mbserver.DiscreteInputsRequest) ([]bool, error) {
	h.log.TRACE.Printf("read discrete: id %d addr %d qty %d", req.UnitId, req.Addr, req.Quantity)
	b, err := h.conn.Clone(req.UnitId).ReadDiscreteInputs(req.Addr, req.Quantity)
	return h.bytesToBoolResult("read discrete", req.Quantity, b, err)
}

func (h *handler) HandleCoils(req *mbserver.CoilsRequest) ([]bool, error) {
	if req.IsWrite {
		switch h.readOnly {
		case ReadOnlyDeny:
			h.log.TRACE.Printf("deny: write coils: id %d addr %d qty %d val %v", req.UnitId, req.Addr, req.Quantity, req.Args)
			return nil, mbserver.ErrIllegalFunction
		case ReadOnlyTrue:
			h.log.TRACE.Printf("ignore: write coils: id %d addr %d qty %d val %v", req.UnitId, req.Addr, req.Quantity, req.Args)
			return req.Args, nil
		}

		if req.WriteFuncCode == gridx.FuncCodeWriteSingleCoil {
			h.log.TRACE.Printf("write coil: id %d addr %d val %t", req.UnitId, req.Addr, req.Args[0])
			var u uint16
			if req.Args[0] {
				u = 0xFF00
			}

			b, err := h.conn.Clone(req.UnitId).WriteSingleCoil(req.Addr, u)
			return h.bytesToBoolResult("write coil", req.Quantity, b, err)
		}

		h.log.TRACE.Printf("write coils: id %d addr %d qty %d val %v", req.UnitId, req.Addr, req.Quantity, req.Args)
		args := coilsToBytes(req.Args)
		b, err := h.conn.Clone(req.UnitId).WriteMultipleCoils(req.Addr, req.Quantity, args)
		return h.bytesToBoolResult("write coils", req.Quantity, b, err)
	}

	h.log.TRACE.Printf("read coils: id %d addr %d qty %d", req.UnitId, req.Addr, req.Quantity)
	b, err := h.conn.Clone(req.UnitId).ReadCoils(req.Addr, req.Quantity)
	return h.bytesToBoolResult("read coils", req.Quantity, b, err)
}

func (h *handler) HandleInputRegisters(req *mbserver.InputRegistersRequest) ([]uint16, error) {
	h.log.TRACE.Printf("read input: id %d addr %d qty %d", req.UnitId, req.Addr, req.Quantity)
	b, err := h.conn.Clone(req.UnitId).ReadInputRegisters(req.Addr, req.Quantity)
	return h.exceptionToUint16AndError("read input", b, err)
}

func (h *handler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	if req.IsWrite {
		switch h.readOnly {
		case ReadOnlyDeny:
			h.log.TRACE.Printf("deny: write holdings: id %d addr %d qty %d val %0x", req.UnitId, req.Addr, req.Quantity, asBytes(req.Args))
			return nil, mbserver.ErrIllegalFunction
		case ReadOnlyTrue:
			h.log.TRACE.Printf("ignore: write holdings: id %d addr %d qty %d val %0x", req.UnitId, req.Addr, req.Quantity, asBytes(req.Args))
			return req.Args, nil
		}

		if req.WriteFuncCode == gridx.FuncCodeWriteSingleRegister {
			h.log.TRACE.Printf("write holding: id %d addr %d val %04x", req.UnitId, req.Addr, req.Args[0])
			b, err := h.conn.Clone(req.UnitId).WriteSingleRegister(req.Addr, req.Args[0])
			return h.exceptionToUint16AndError("write holding", b, err)
		}

		h.log.TRACE.Printf("write holdings: id %d addr %d qty %d val %0x", req.UnitId, req.Addr, req.Quantity, asBytes(req.Args))
		b, err := h.conn.Clone(req.UnitId).WriteMultipleRegisters(req.Addr, req.Quantity, asBytes(req.Args))
		return h.exceptionToUint16AndError("write multiple holding", b, err)
	}

	h.log.TRACE.Printf("read holdings: id %d addr %d qty %d", req.UnitId, req.Addr, req.Quantity)
	b, err := h.conn.Clone(req.UnitId).ReadHoldingRegisters(req.Addr, req.Quantity)
	return h.exceptionToUint16AndError("read holding", b, err)
}
