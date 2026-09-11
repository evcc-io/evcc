package charger

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
	"encoding/binary"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util/modbus"
)

const (
	myPvRegPower          = 1000
	myPvRegTempLimit      = 1002
	myPvRegOperationState = 1077
)

type myPvBase struct {
	embed
	conn    *modbus.Connection
	lp      loadpoint.API
	regTemp uint16
}

func newMyPvBase(conn *modbus.Connection, regTemp uint16) myPvBase {
	return myPvBase{
		embed: embed{
			Icon_:     "waterheater",
			Features_: []api.Feature{api.IntegratedDevice, api.Heating},
		},
		conn:    conn,
		regTemp: regTemp,
	}
}

func (wb *myPvBase) readUint16(reg uint16) (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 1)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

var _ api.Battery = (*myPvBase)(nil)

// Soc implements the api.Battery interface (returns water temperature).
func (wb *myPvBase) Soc() (float64, error) {
	v, err := wb.readUint16(wb.regTemp)
	return float64(v) / 10, err
}

var _ api.SocLimiter = (*myPvBase)(nil)

// GetLimitSoc implements the api.SocLimiter interface.
func (wb *myPvBase) GetLimitSoc() (int64, error) {
	v, err := wb.readUint16(myPvRegTempLimit)
	return int64(v) / 10, err
}

var _ loadpoint.Controller = (*myPvBase)(nil)

// LoadpointControl implements loadpoint.Controller.
func (wb *myPvBase) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
