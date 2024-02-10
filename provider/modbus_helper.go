package provider

import (
	"encoding/binary"
	"fmt"
)

// UintFromBytes converts byte slice to bigendian uint value
func UintFromBytes(bytes []byte) (u uint64, err error) {
	switch l := len(bytes); l {
	case 1:
		u = uint64(bytes[0])
	case 2:
		u = uint64(binary.BigEndian.Uint16(bytes))
	case 4:
		u = uint64(binary.BigEndian.Uint32(bytes))
	case 8:
		u = binary.BigEndian.Uint64(bytes)
	default:
		err = fmt.Errorf("unexpected length: %d", l)
	}

	return u, err
}
