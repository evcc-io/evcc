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

func (r Register) FuncCode() (uint8, error) {
	switch strings.ToLower(r.Type) {
	case "holding":
		return modbus.FuncCodeReadHoldingRegisters, nil
	case "input":
		return modbus.FuncCodeReadInputRegisters, nil
	case "coil":
		return modbus.FuncCodeReadCoils, nil
	case "writesingle", "writeholding":
		return modbus.FuncCodeWriteSingleRegister, nil
	case "writemultiple", "writeholdings":
		return modbus.FuncCodeWriteMultipleRegisters, nil
	case "writecoil":
		return modbus.FuncCodeWriteSingleCoil, nil
	default:
		return 0, fmt.Errorf("invalid register type: %s", r.Type)
	}
}

func (r Register) DecodeFunc() (func([]byte) float64, error) {
	switch strings.ToLower(r.encoding()) {
	// 8 bit (coil)
	case "bool8":
		return decodeBool8, nil

	// 16 bit
	case "int16":
		return asFloat64(encoding.Int16), nil
	case "int16nan":
		return decodeNaN16(asFloat64(encoding.Int16), 1<<15, 1<<15-1), nil
	case "uint16":
		return asFloat64(encoding.Uint16), nil
	case "uint16nan":
		return decodeNaN16(asFloat64(encoding.Uint16), 1<<16-1), nil
	case "bool16":
		mask, err := decodeMask(r.BitMask)
		if err != nil {
			return nil, err
		}
		return decodeBool16(mask), nil

	// 32 bit
	case "int32":
		return asFloat64(encoding.Int32), nil
	case "int32nan":
		return decodeNaN32(asFloat64(encoding.Int32), 1<<31, 1<<31-1), nil
	case "int32s":
		return asFloat64(encoding.Int32LswFirst), nil
	case "uint32":
		return asFloat64(encoding.Uint32), nil
	case "uint32s":
		return asFloat64(encoding.Uint32LswFirst), nil
	case "uint32nan":
		return decodeNaN32(asFloat64(encoding.Uint32), 1<<32-1), nil
	case "float32", "ieee754":
		return asFloat64(encoding.Float32), nil
	case "float32s", "ieee754s":
		return asFloat64(encoding.Float32LswFirst), nil

	// 64 bit
	case "uint64":
		return asFloat64(encoding.Uint64), nil
	case "uint64nan":
		return decodeNaN64(asFloat64(encoding.Uint64), 1<<64-1), nil
	case "float64":
		return encoding.Float64, nil

	default:
		return nil, fmt.Errorf("invalid register decoding: %s", r.Decode)
	}
}

func (r Register) isFloat32() bool {
	return slices.Contains([]string{
		"float32", "float32s",
		"ieee754", "ieee754s",
	}, strings.ToLower(r.encoding()))
}

func (r Register) isFloat64() bool {
	return slices.Contains([]string{
		"float64", "float64s",
	}, strings.ToLower(r.encoding()))
}

// Operation creates a modbus operation from a register definition
func (r Register) Operation() (RegisterOperation, error) {
	len, err := r.Length()
	if err != nil {
		return RegisterOperation{}, err
	}

	fc, err := r.FuncCode()
	if err != nil {
		return RegisterOperation{}, err
	}

	op := RegisterOperation{
		Addr:     r.Address,
		Length:   len,
		FuncCode: fc,
	}

	if op.IsRead() {
		op.Decode, err = r.DecodeFunc()
		if err != nil {
			return RegisterOperation{}, err
		}
	} else {
		switch strings.ToLower(r.encoding()) {
		case "int32s", "uint32s",
			"float32s", "ieee754s":
			op.Encode = func(v uint64) uint64 {
				fmt.Printf("swapped:\t% x\n", v)
				return v&0xFFFF<<16 | v&0xFFFF0000>>16
			}

		// case "float32s", "ieee754s":
		// 	op.Encode = func(v uint64) uint64 {
		// 		fmt.Printf("litte endian: % x\n", v)
		// 		b := make([]byte, 4)
		// 		encoding.PutFloat32LswFirst(b, float32(v))
		// 		return uint64(binary.BigEndian.Uint32(b))
		// 	}

		default:
			op.Encode = func(v uint64) uint64 {
				fmt.Printf("big endian:\t% x\n", v)
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
