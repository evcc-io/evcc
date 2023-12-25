package modbus

type Encoding int

//go:generate enumer -type Encoding -transform=lower
const (
	_ Encoding = iota

	Bool8
	Bool16

	Int16
	Int32
	Int64

	Int16NaN
	Int32NaN
	Int64NaN
	Int32S

	Uint16
	Uint32
	Uint64

	Uint16NaN
	Uint32NaN
	Uint64NaN
	Uint32S

	Float32
	Float32S
	Float64
)

// Len is the number of words
func (e Encoding) Len() uint16 {
	switch e {
	case Bool8:
		return 1
	case Bool16:
		return 1
	case Int16:
		return 1
	case Int32:
		return 2
	case Int64:
		return 4
	case Int16NaN:
		return 1
	case Int32NaN:
		return 2
	case Int64NaN:
		return 4
	case Int32S:
		return 2
	case Uint16:
		return 1
	case Uint32:
		return 2
	case Uint64:
		return 4
	case Uint16NaN:
		return 1
	case Uint32NaN:
		return 2
	case Uint64NaN:
		return 4
	case Uint32S:
		return 2
	case Float32:
		return 2
	case Float32S:
		return 2
	case Float64:
		return 4
	default:
		return 0
	}
}

// creates a read operation from a register definition
func (e Encoding) Transform() func([]byte) float64 {
	return nil

	// switch e {
	// // 8 bit (coil)
	// case Bool8:
	// 	return decodeBool8

	// // 16 bit
	// case Int16:
	// 	return asFloat64(encoding.Int16)
	// case Int16NaN:
	// 	return decodeNaN16(asFloat64(encoding.Int16), 1<<15, 1<<15-1)
	// case Uint16:
	// 	return asFloat64(encoding.Uint16)
	// case Uint16NaN:
	// 	return decodeNaN16(asFloat64(encoding.Uint16), 1<<16-1)
	// case Bool16:
	// 	mask, err := decodeMask(r.BitMask)
	// 	if err != nil {
	// 		return op, err
	// 	}
	// 	return decodeBool16(mask)

	// // 32 bit
	// case Int32:
	// 	return asFloat64(encoding.Int32)
	// case Int32NaN:
	// 	return decodeNaN32(asFloat64(encoding.Int32), 1<<31, 1<<31-1)
	// case Int32S:
	// 	return asFloat64(encoding.Int32LswFirst)
	// case Uint32:
	// 	return asFloat64(encoding.Uint32)
	// case Uint32S:
	// 	return asFloat64(encoding.Uint32LswFirst)
	// case Uint32NaN:
	// 	return decodeNaN32(asFloat64(encoding.Uint32), 1<<32-1)
	// case Float32:
	// 	return asFloat64(encoding.Float32)
	// case Float32S:
	// 	return asFloat64(encoding.Float32LswFirst)

	// // 64 bit
	// case Uint64:
	// 	return asFloat64(encoding.Uint64)
	// case Uint64NaN:
	// 	return decodeNaN64(asFloat64(encoding.Uint64), 1<<64-1)
	// case Float64:
	// 	return encoding.Float64

	// default:
	// 	return nil, fmt.Errorf("invalid register decoding: %s", r.Decode)
	// }

	// return op, nil
}
