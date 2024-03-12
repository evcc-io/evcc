package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
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
	case "int64":
		return asFloat64(encoding.Int64), nil
	case "uint64":
		return asFloat64(encoding.Uint64), nil
	case "uint64nan":
		return decodeNaN64(asFloat64(encoding.Uint64), 1<<64-1), nil
	case "float64":
		return encoding.Float64, nil

	default:
		return nil, fmt.Errorf("invalid register decoding: %s", r.encoding())
	}
}

func (r Register) encodeToBytes(fun func(float64) uint64) (func(float64) ([]byte, error), error) {
	length, err := r.Length()
	if err != nil {
		return nil, err
	}

	// swapped
	if strings.HasSuffix(strings.ToLower(r.encoding()), "s") {
		if length != 2 {
			return nil, fmt.Errorf("invalid swapped encoding register length: %d", length)
		}

		inner := fun
		fun = func(f float64) uint64 {
			v := inner(f)
			return v&0xFFFF<<16 | v&0xFFFF0000>>16
		}
	}

	return func(f float64) ([]byte, error) {
		v := fun(f)
		b := make([]byte, 2*length)

		switch length {
		case 1:
			binary.BigEndian.PutUint16(b[:], uint16(v))
		case 2:
			binary.BigEndian.PutUint32(b[:], uint32(v))
		case 4:
			binary.BigEndian.PutUint64(b[:], v)
		default:
			return nil, fmt.Errorf("invalid register length: %d", length)
		}

		return b, nil
	}, nil
}

func (r Register) EncodeFunc() (func(float64) ([]byte, error), error) {
	enc := strings.ToLower(r.encoding())

	switch {
	case strings.HasPrefix(enc, "int") || strings.HasPrefix(enc, "uint"):
		return r.encodeToBytes(func(v float64) uint64 {
			return uint64(v)
		})

	case strings.HasPrefix(enc, "float") || strings.HasPrefix(enc, "ieee754"):
		length, err := r.Length()
		if err != nil {
			return nil, err
		}

		switch length {
		case 2:
			return r.encodeToBytes(func(v float64) uint64 {
				return uint64(math.Float32bits(float32(v)))
			})

		case 4:
			return r.encodeToBytes(func(v float64) uint64 {
				return math.Float64bits(v)
			})

		default:
			return nil, fmt.Errorf("invalid register length: %d", length)
		}

	default:
		return nil, fmt.Errorf("invalid register encoding: %s", r.encoding())
	}
}

type RegisterOperation struct {
	FuncCode uint8
	Addr     uint16
	Length   uint16
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
