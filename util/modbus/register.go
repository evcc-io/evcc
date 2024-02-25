package modbus

import (
	"encoding/binary"
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

func (r Register) Length() (uint16, error) {
	enc := r.encoding()
	switch {
	case strings.Contains(enc, "8") || strings.Contains(enc, "16"):
		return 1, nil
	case strings.Contains(enc, "32") || strings.Contains(enc, "754"):
		return 2, nil
	case strings.Contains(enc, "64"):
		return 4, nil
	default:
		return 0, fmt.Errorf("invalid register encoding: %s", enc)
	}
}

// Operation creates a modbus operation from a register definition
func (r Register) Operation() (RegisterOperation, error) {
	len, err := r.Length()
	if err != nil {
		return RegisterOperation{}, err
	}

	op := RegisterOperation{
		Addr:   r.Address,
		Length: len,
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

		// 16 bit
		case "int16":
			op.Decode = asFloat64(encoding.Int16)
		case "int16nan":
			op.Decode = decodeNaN16(asFloat64(encoding.Int16), 1<<15, 1<<15-1)
		case "uint16":
			op.Decode = asFloat64(encoding.Uint16)
		case "uint16nan":
			op.Decode = decodeNaN16(asFloat64(encoding.Uint16), 1<<16-1)
		case "bool16":
			mask, err := decodeMask(r.BitMask)
			if err != nil {
				return op, err
			}
			op.Decode = decodeBool16(mask)

		// 32 bit
		case "int32":
			op.Decode = asFloat64(encoding.Int32)
		case "int32nan":
			op.Decode = decodeNaN32(asFloat64(encoding.Int32), 1<<31, 1<<31-1)
		case "int32s":
			op.Decode = asFloat64(encoding.Int32LswFirst)
		case "uint32":
			op.Decode = asFloat64(encoding.Uint32)
		case "uint32s":
			op.Decode = asFloat64(encoding.Uint32LswFirst)
		case "uint32nan":
			op.Decode = decodeNaN32(asFloat64(encoding.Uint32), 1<<32-1)
		case "float32", "ieee754":
			op.Decode = asFloat64(encoding.Float32)
		case "float32s", "ieee754s":
			op.Decode = asFloat64(encoding.Float32LswFirst)

		// 64 bit
		case "uint64":
			op.Decode = asFloat64(encoding.Uint64)
		case "uint64nan":
			op.Decode = decodeNaN64(asFloat64(encoding.Uint64), 1<<64-1)
		case "float64":
			op.Decode = encoding.Float64

		default:
			return RegisterOperation{}, fmt.Errorf("invalid register decoding: %s", r.Decode)
		}
	} else {
		switch strings.ToLower(r.encoding()) {
		case "int32s", "uint32s":
			op.Encode = func(v uint64) uint64 {
				return v&0xFFFF<<16 | v&0xFFFF0000>>16
			}

		case "float32s", "ieee754s":
			op.Encode = func(v uint64) uint64 {
				b := make([]byte, 4)
				encoding.PutFloat32LswFirst(b, float32(v))
				return uint64(binary.BigEndian.Uint32(b))
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
