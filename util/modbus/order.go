package modbus

import (
	"encoding/binary"
	"strings"
)

const networkOrder = "ABCDEFGH"

func Ordered(order string, b []byte) []byte {
	if len(order) != len(b) {
		return nil
	}

	order = strings.ToUpper(order)

	if strings.HasPrefix(networkOrder, order) {
		return b
	}

	res := make([]byte, len(b))
	for i, c := range order {
		pos := c - 'A'
		res[pos] = b[i]
	}

	return res
}

func Ordered16(order string, b []byte) uint16 {
	return binary.BigEndian.Uint16(Ordered(order, b))
}

func Ordered32(order string, b []byte) uint32 {
	return binary.BigEndian.Uint32(Ordered(order, b))
}

func Ordered64(order string, b []byte) uint64 {
	return binary.BigEndian.Uint64(Ordered(order, b))
}
