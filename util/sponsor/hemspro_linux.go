//go:build linux

package sponsor

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"os"
	"strings"

	i2c "github.com/d2r2/go-i2c"
)

const hemspro = "hemspro"

// deviceSerial returns the device tree serial number or empty string
func deviceSerial() string {
	b, err := os.ReadFile("/sys/firmware/devicetree/base/serial-number")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Trim(string(b), "\x00"))
}

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

	// I2C succeeded — verify with server
	return checkHardware(hemspro, map[string]string{
		"serial": deviceSerial(),
	})
}

func checkHemsPro() (string, string) {
	const (
		ADDR         = 0b1101000 // 0x68 DS1307
		REG_TIMEDATE = 0x00
	)

	// Bus 1 is the Raspberry Pi's general-purpose header I2C bus (I2C0 is
	// reserved for HAT-EEPROM identification) - true for every RPi model.
	// Other SBCs number their I2C controllers by hardware instance with no
	// such convention; on a Banana Pi BPI-M2 Zero for example, the header's
	// I2C lands on bus 0 while bus 1 is the SoC's internal HDMI DDC bus.
	for _, bus := range []int{1, 0} {
		i2c, err := i2c.NewI2C(ADDR, bus)
		if err != nil {
		        continue
		}
		
		if _, err := i2c.WriteBytes([]byte{REG_TIMEDATE}); err != nil {
		        i2c.Close()
		        continue
		}
		
		buf := make([]byte, 7)
		n, err := i2c.ReadBytes(buf)
		i2c.Close()
		if err != nil || n != 7 {
		        continue
		}
		
		// I2C succeeded — verify with server
		return checkHardware(hemspro, map[string]string{
	        "serial": deviceSerial(),
	})
}

return "", ""
  }

