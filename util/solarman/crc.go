package solarman

func CRC(b []byte) (byte, byte) {
	var crc uint16 = 0xFFFF

	for _, v := range b {
		crc = crc ^ uint16(v)
		for range 8 {
			if (crc & 0x01) == 0 {
				crc = crc >> 1
			} else {
				crc = crc >> 1
				crc = crc ^ 0xA001
			}
		}
	}
	lo := byte(crc)
	hi := byte(crc >> 8)
	return lo, hi
}
