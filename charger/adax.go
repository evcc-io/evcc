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
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/charger/adax"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
)

// Adax controls the heaters of an Adax room by switching heating and its target temperature
type Adax struct {
	*SgReady
}

func init() {
	registry.AddCtx("adax", NewAdaxFromConfig)
}

// the Adax api accepts target temperatures from 5 to 35 °C
const adaxMinTemp, adaxMaxTemp = 5.0, 35.0

// NewAdaxFromConfig creates an Adax heater charger from generic config
func NewAdaxFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed    `mapstructure:",squash"`
		User     string
		Password string
		Room     string
		Normal   float64 // target temperature outside boost
		Setpoint float64 // target temperature during boost
		Power    int64   // rated heater power used to estimate consumption
		Cache    time.Duration
	}{
		embed: embed{
			Icon_:     "heater",
			Features_: []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice, api.SwitchDevice},
		},
		Normal:   20,
		Setpoint: 22,
		Cache:    5 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	// positive form so that NaN fails validation
	if !(cc.Normal >= adaxMinTemp && cc.Setpoint <= adaxMaxTemp && cc.Setpoint > cc.Normal) {
		return nil, fmt.Errorf("invalid temperatures: boost setpoint %.1f must exceed normal %.1f, both within %.0f..%.0f", cc.Setpoint, cc.Normal, adaxMinTemp, adaxMaxTemp)
	}

	log := util.NewLogger("adax").Redact(cc.Password)

	ts, err := adax.TokenSource(ctx, log, request.NewClient(log), cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	conn := adax.NewAPI(log, ts)

	rooms, err := conn.Rooms(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get rooms: %w", err)
	}

	room, err := adaxRoom(rooms, cc.Room)
	if err != nil {
		return nil, err
	}

	// commands only show up in the api after minutes, so there is no point resetting the cache after control
	roomG := util.Cached(func() (adax.Room, error) {
		rooms, err := conn.Rooms(ctx)
		if err != nil {
			return adax.Room{}, err
		}
		return adaxRoom(rooms, strconv.FormatInt(room.Id, 10))
	}, cc.Cache)

	res := new(Adax)

	target := func(mode int64) float64 {
		return map[int64]float64{Normal: cc.Normal, Boost: cc.Setpoint}[mode]
	}

	// mode set by evcc, 0 until evcc first sets a mode
	var (
		mu      sync.Mutex
		setMode int64
	)

	set := func(mode int64) error {
		if mode != Dim && mode != Normal && mode != Boost {
			return api.ErrNotAvailable
		}

		mu.Lock()
		defer mu.Unlock()

		t := target(mode)
		if err := conn.SetHeating(ctx, room.Id, t > 0, t); err != nil {
			return err
		}

		setMode = mode

		return nil
	}

	// the set mode takes precedence over the room's reported state, which lags behind control commands by several minutes
	getMode := func() (int64, error) {
		mu.Lock()
		mode := setMode
		mu.Unlock()

		if mode != 0 {
			return mode, nil
		}

		r, err := roomG()
		if err != nil {
			return 0, err
		}

		return adaxMode(r, cc.Setpoint), nil
	}

	res.SgReady, err = NewSgReady(ctx, &cc.embed, set, getMode, nil)
	if err != nil {
		return nil, err
	}

	implement.Has(res, implement.Battery(func() (float64, error) {
		r, err := roomG()
		if err != nil {
			return 0, err
		}

		temp, ok := r.CurrentTemperature()
		if !ok {
			return 0, api.ErrNotAvailable
		}

		return temp, nil
	}))

	// the api only reports energy from a delayed snapshot, so power is estimated from the target heating state
	if cc.Power > 0 {
		implement.Has(res, implement.Meter(func() (float64, error) {
			r, err := roomG()
			if err != nil {
				return 0, err
			}

			mu.Lock()
			mode := setMode
			mu.Unlock()

			var setTarget *float64
			if mode != 0 {
				t := target(mode)
				setTarget = &t
			}

			return adaxPower(r, setTarget, float64(cc.Power))
		}))
	}

	return res, nil
}

// adaxMode derives the mode from the room's reported state
func adaxMode(r adax.Room, setpoint float64) int64 {
	if r.HeatingEnabled != nil && !*r.HeatingEnabled {
		return Dim
	}

	if t, ok := r.TargetCelsius(); ok && t >= setpoint {
		return Boost
	}

	return Normal
}

// adaxPower estimates power as rated power while the room is below target, using the target evcc has set if any
func adaxPower(r adax.Room, setTarget *float64, power float64) (float64, error) {
	temp, ok := r.CurrentTemperature()
	if !ok {
		return 0, api.ErrNotAvailable
	}

	var target float64
	switch {
	case setTarget != nil:
		target = *setTarget
	case r.HeatingEnabled != nil && !*r.HeatingEnabled:
	default:
		if target, ok = r.TargetCelsius(); !ok {
			return 0, api.ErrNotAvailable
		}
	}

	if target == 0 || temp >= target {
		return 0, nil
	}

	return power, nil
}

// adaxRoom selects the room by id or unique name, or the only room if none is configured
func adaxRoom(rooms []adax.Room, id string) (adax.Room, error) {
	if id == "" && len(rooms) == 1 {
		return rooms[0], nil
	}

	if r, ok := lo.Find(rooms, func(r adax.Room) bool {
		return strconv.FormatInt(r.Id, 10) == id
	}); ok {
		return r, nil
	}

	matches := lo.Filter(rooms, func(r adax.Room, _ int) bool {
		return id != "" && strings.EqualFold(r.Name, id)
	})

	if len(matches) == 1 {
		return matches[0], nil
	}

	names := strings.Join(lo.Map(rooms, func(r adax.Room, _ int) string {
		return fmt.Sprintf("%d (%s)", r.Id, r.Name)
	}), ", ")

	if len(matches) > 1 {
		return adax.Room{}, fmt.Errorf("room name %q is ambiguous, use the id: %s", id, names)
	}

	return adax.Room{}, fmt.Errorf("cannot find room, got: %s", names)
}
