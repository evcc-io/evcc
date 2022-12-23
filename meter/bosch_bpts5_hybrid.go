package meter

// LICENSE

// Bosch is the Bosch BPT-S 5 Hybrid meter

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"errors"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/bosch"
	"github.com/evcc-io/evcc/util"
)

// Example config:
// meters:
// - name: bosch_grid
//   type: bosch-bpt
//   uri: http://192.168.178.22
//   usage: grid
// - name: bosch_pv
//   type: bosch-bpt
//   uri: http://192.168.178.22
//   usage: pv
// - name: bosch_battery
//   type: bosch-bpt
//   uri: http://192.168.178.22
//   usage: battery

type BoschBpts5Hybrid struct {
	api   *bosch.API
	usage string
}

func init() {
	registry.Add("bosch-bpt", NewBoschBpts5HybridFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateBoschBpts5Hybrid -b api.Meter -t "api.Battery,Soc,func() (float64, error)"

// NewBoschBpts5HybridFromConfig creates a Bosch BPT-S 5 Hybrid Meter from generic config
func NewBoschBpts5HybridFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		URI   string
		Usage string
		Cache time.Duration
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	return NewBoschBpts5Hybrid(cc.URI, cc.Usage, cc.Cache)
}

// NewBoschBpts5Hybrid creates a Bosch BPT-S 5 Hybrid Meter
func NewBoschBpts5Hybrid(uri, usage string, cache time.Duration) (api.Meter, error) {
	log := util.NewLogger("bosch-bpt")

	instance, exists := bosch.Instances.LoadOrStore(uri, bosch.NewLocal(log, uri, cache))
	if !exists {
		if err := instance.(*bosch.API).Login(); err != nil {
			return nil, err
		}
	}

	m := &BoschBpts5Hybrid{
		api:   instance.(*bosch.API),
		usage: strings.ToLower(usage),
	}

	// decorate api.BatterySoc
	var batterySoc func() (float64, error)
	if usage == "battery" {
		batterySoc = m.batterySoc
	}

	return decorateBoschBpts5Hybrid(m, batterySoc), nil
}

// CurrentPower implements the api.Meter interface
func (m *BoschBpts5Hybrid) CurrentPower() (float64, error) {
	status, err := m.api.Status()

	switch m.usage {
	case "grid":
		return status.BuyFromGrid - status.SellToGrid, err
	case "pv":
		return status.PvPower, err
	case "battery":
		return status.BatteryDischargePower - status.BatteryChargePower, err
	default:
		return 0, err
	}
}

// batterySoc implements the api.Battery interface
func (m *BoschBpts5Hybrid) batterySoc() (float64, error) {
	status, err := m.api.Status()
	return status.CurrentBatterySoc, err
}
