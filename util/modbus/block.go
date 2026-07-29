package modbus

import "fmt"

// Block describes an enclosing register block: Count 16-bit registers starting
// at Register, fetched once per poll cycle and shared via a Cache.
type Block struct {
	Register uint16
	Count    uint16
}

// Contains reports whether the count registers starting at register fit
// entirely within the block.
func (b Block) Contains(register, count uint16) bool {
	return register >= b.Register &&
		uint32(register)+uint32(count) <= uint32(b.Register)+uint32(b.Count)
}

// ByteOffset returns the byte offset of register within the block payload
// (each register occupies two bytes).
func (b Block) ByteOffset(register uint16) int {
	return int(register-b.Register) * 2
}

// Extract returns the bytes for op sliced out of a block payload.
func (b Block) Extract(op RegisterOperation, payload []byte) ([]byte, error) {
	if !b.Contains(op.Addr, op.Length) {
		return nil, fmt.Errorf("register %d+%d does not fit in block %d+%d", op.Addr, op.Length, b.Register, b.Count)
	}

	offset := b.ByteOffset(op.Addr)
	end := offset + int(op.Length)*2
	if len(payload) < end {
		return nil, fmt.Errorf("block payload too short (len=%d, need=%d)", len(payload), end)
	}

	return payload[offset:end], nil
}
