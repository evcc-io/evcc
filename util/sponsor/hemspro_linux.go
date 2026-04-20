//go:build linux

package sponsor

import (
	i2c "github.com/d2r2/go-i2c"
)

const hemspro = "hemspro"

// checkHemsPro checks if the hardware is a supported HEMS Pro device and returns sponsor subject
func checkHemsPro() string {
	const (
		ADDR         = 0b1101000 // 0x68 DS1307
		REG_TIMEDATE = 0x00
	)

	// Create new connection to I2C bus 1
	i2c, err := i2c.NewI2C(ADDR, 1)
	if err != nil {
		return ""
	}
	defer i2c.Close()

	if _, err := i2c.WriteBytes([]byte{REG_TIMEDATE}); err != nil {
		return ""
	}

	buf := make([]byte, 7)
	if n, err := i2c.ReadBytes(buf); err != nil || n != 7 {
		return ""
	}

	return hemspro
}
