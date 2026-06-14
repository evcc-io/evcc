// Package blockread provides transport-agnostic block-read primitives shared by
// the modbus and aa55 source plugins: fetch one enclosing block, extract many.
package blockread

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
