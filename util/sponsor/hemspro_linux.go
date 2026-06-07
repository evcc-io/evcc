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

	// I2C succeeded — verify with server
	return checkHardware(hemspro, nil)
}
