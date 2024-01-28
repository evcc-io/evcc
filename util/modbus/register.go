package modbus

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/grid-x/modbus"
	"github.com/volkszaehler/mbmd/encoding"
	"golang.org/x/exp/constraints"
)

// Register contains the ModBus register configuration
type Register struct {
	Address  uint16 // Length  uint16
	Type     string
	Decode   string // TODO deprecated, use Encoding
	Encoding string
	BitMask  string
}

func (r Register) Error() error {
	if r.Address == 0 {
		return errors.New("address is required")
	}
	if r.Type == "" {
		return errors.New("type is required")
	}
	if r.Decode == "" && r.Encoding == "" {
		return errors.New("encoding is required")
	}
	if r.Decode != "" && r.Encoding != "" {
		return errors.New("must not have decide when encoding is specified")
	}
	return nil
}

func (r Register) encoding() string {
	if r.Encoding != "" {
		return r.Encoding
	}
	return r.Decode
}

// Operation creates a modbus operation from a register definition
func (r Register) Operation() (RegisterOperation, error) {
	op := RegisterOperation{
		Addr: r.Address,
	}

	switch strings.ToLower(r.Type) {
	case "holding":
		op.FuncCode = modbus.FuncCodeReadHoldingRegisters
	case "input":
		op.FuncCode = modbus.FuncCodeReadInputRegisters
	case "coil":
		op.FuncCode = modbus.FuncCodeReadCoils
	case "writesingle", "writeholding":
		op.FuncCode = modbus.FuncCodeWriteSingleRegister
	case "writemultiple", "writeholdings":
		op.FuncCode = modbus.FuncCodeWriteMultipleRegisters
	case "writecoil":
		op.FuncCode = modbus.FuncCodeWriteSingleCoil
	default:
		return RegisterOperation{}, fmt.Errorf("invalid register type: %s", r.Type)
	}

	if op.IsRead() {
		switch strings.ToLower(r.encoding()) {
		// 8 bit (coil)
		case "bool8":
			op.Decode = decodeBool8
			op.Length = 1

		// 16 bit
		case "int16":
			op.Decode = asFloat64(encoding.Int16)
			op.Length = 1
		case "int16nan":
			op.Decode = decodeNaN16(asFloat64(encoding.Int16), 1<<15, 1<<15-1)
			op.Length = 1
		case "uint16":
			op.Decode = asFloat64(encoding.Uint16)
			op.Length = 1
		case "uint16nan":
			op.Decode = decodeNaN16(asFloat64(encoding.Uint16), 1<<16-1)
			op.Length = 1
		case "bool16":
			mask, err := decodeMask(r.BitMask)
			if err != nil {
				return op, err
			}
			op.Decode = decodeBool16(mask)
			op.Length = 1

		// 32 bit
		case "int32":
			op.Decode = asFloat64(encoding.Int32)
			op.Length = 2
		case "int32nan":
			op.Decode = decodeNaN32(asFloat64(encoding.Int32), 1<<31, 1<<31-1)
			op.Length = 2
		case "int32s":
			op.Decode = asFloat64(encoding.Int32LswFirst)
			op.Length = 2
		case "uint32":
			op.Decode = asFloat64(encoding.Uint32)
			op.Length = 2
		case "uint32s":
			op.Decode = asFloat64(encoding.Uint32LswFirst)
			op.Length = 2
		case "uint32nan":
			op.Decode = decodeNaN32(asFloat64(encoding.Uint32), 1<<32-1)
			op.Length = 2
		case "float32", "ieee754":
			op.Decode = asFloat64(encoding.Float32)
			op.Length = 2
		case "float32s", "ieee754s":
			op.Decode = asFloat64(encoding.Float32LswFirst)
			op.Length = 2

		// 64 bit
		case "uint64":
			op.Decode = asFloat64(encoding.Uint64)
			op.Length = 4
		case "uint64nan":
			op.Decode = decodeNaN64(asFloat64(encoding.Uint64), 1<<64-1)
			op.Length = 4
		case "float64":
			op.Decode = encoding.Float64
			op.Length = 4

		default:
			return RegisterOperation{}, fmt.Errorf("invalid register decoding: %s", r.Decode)
		}
	} else {
		switch strings.ToLower(r.encoding()) {
		case "int32s", "uint32s", "float32s", "ieee754s":
			op.Encode = func(v uint64) uint64 {
				return v&0xFFFF<<16 | v&0xFFFF0000>>16
			}

		default:
			op.Encode = func(v uint64) uint64 {
				return v
			}
		}
	}

	return op, nil
}

// asFloat64 creates a function that returns numerics vales as float64
func asFloat64[T constraints.Signed | constraints.Unsigned | constraints.Float](f func([]byte) T) func([]byte) float64 {
	return func(v []byte) float64 {
		res := float64(f(v))
		if math.IsNaN(res) || math.IsInf(res, 0) {
			res = 0
		}
		return res
	}
}

type RegisterOperation struct {
	FuncCode uint8
	Addr     uint16
	Length   uint16
	Encode   func(uint64) uint64
	Decode   func([]byte) float64
}

func (op RegisterOperation) IsRead() bool {
	return !slices.Contains([]uint8{
		modbus.FuncCodeWriteSingleRegister,
		modbus.FuncCodeWriteMultipleRegisters,
		modbus.FuncCodeWriteSingleCoil,
	}, op.FuncCode)
}
