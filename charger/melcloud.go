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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.AddCtx("melcloud", NewMelcloudFromConfig)
}

// MELCloud API
const (
	melcloudURI        = "https://app.melcloud.com/Mitsubishi.Wifi.Client"
	melcloudAppVersion = "1.34.7.0"

	// device types
	melcloudATA = 0 // Air-to-Air (e.g. AC)
	melcloudATW = 1 // Air-to-Water (e.g. heat pump)

	// EffectiveFlags - ATA
	melcloudFlagAtaPower          = 0x01
	melcloudFlagAtaSetTemperature = 0x04

	// EffectiveFlags - ATW
	melcloudFlagAtwPower               = 0x01
	melcloudFlagAtwSetTemperatureZone1 = 0x200000080
	melcloudFlagAtwSetTankTemperature  = 0x1000000000020
)

// Melcloud charger implementation
type Melcloud struct {
	*SgReady
	*request.Helper
	log        *util.Logger
	mu         sync.Mutex
	user       string
	password   string
	contextKey string
	deviceID   int64
	buildingID int64
	deviceType int
	useTank    bool
	normalTemp float64
	boostTemp  float64
}

// NewMelcloudFromConfig creates a MELCloud configurable charger from generic config
func NewMelcloudFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		Device         string
		TempSource     string
		NormalTemp     float64
		BoostTemp      float64
		Cache          time.Duration
	}{
		embed: embed{
			Icon_:     "heatpump",
			Features_: []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice, api.SwitchDevice},
		},
		NormalTemp: 45,
		BoostTemp:  60,
		Cache:      time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.BoostTemp <= cc.NormalTemp {
		return nil, errors.New("boostTemp must be greater than normalTemp")
	}

	log := util.NewLogger("melcloud").Redact(cc.User, cc.Password)

	m := &Melcloud{
		Helper:     request.NewHelper(log),
		log:        log,
		user:       cc.User,
		password:   cc.Password,
		useTank:    cc.TempSource == "warmwater",
		normalTemp: cc.NormalTemp,
		boostTemp:  cc.BoostTemp,
	}

	if err := m.login(); err != nil {
		return nil, err
	}

	dev, err := m.selectDevice(cc.Device)
	if err != nil {
		return nil, err
	}
	m.deviceID = dev.DeviceID
	m.buildingID = dev.BuildingID
	m.deviceType = dev.Device.DeviceType

	sgr, err := NewSgReady(ctx, &cc.embed, m.setMode, util.Cached(m.getMode, cc.Cache), nil)
	if err != nil {
		return nil, err
	}
	m.SgReady = sgr

	implement.Has(m, implement.Battery(util.Cached(m.temperature, cc.Cache)))

	return m, nil
}

// MELCloud API types

type melcloudLoginRequest struct {
	Email           string  `json:"Email"`
	Password        string  `json:"Password"`
	Language        int     `json:"Language"`
	AppVersion      string  `json:"AppVersion"`
	Persist         bool    `json:"Persist"`
	CaptchaResponse *string `json:"CaptchaResponse"`
}

type melcloudLoginResponse struct {
	ErrorId      *int
	ErrorMessage string
	LoginData    *struct {
		ContextKey string
		Name       string
	}
}

type melcloudListDevice struct {
	DeviceID   int64
	BuildingID int64
	DeviceName string
	Device     struct {
		DeviceType int
	}
}

type melcloudArea struct {
	Devices []melcloudListDevice
}

type melcloudFloor struct {
	Areas   []melcloudArea
	Devices []melcloudListDevice
}

type melcloudBuilding struct {
	ID        int64
	Structure struct {
		Devices []melcloudListDevice
		Areas   []melcloudArea
		Floors  []melcloudFloor
	}
}

// melcloudDeviceState is the union of fields used by both ATA and ATW.
// Unknown fields are preserved by round-tripping through a generic map.
type melcloudDeviceState map[string]any

// login authenticates with MELCloud and stores the context key
func (m *Melcloud) login() error {
	body := melcloudLoginRequest{
		Email:      m.user,
		Password:   m.password,
		Language:   0,
		AppVersion: melcloudAppVersion,
		Persist:    true,
	}

	req, err := request.New(http.MethodPost, melcloudURI+"/Login/ClientLogin", request.MarshalJSON(body), request.JSONEncoding)
	if err != nil {
		return err
	}

	var res melcloudLoginResponse
	if err := m.DoJSON(req, &res); err != nil {
		return err
	}

	if res.LoginData == nil || res.LoginData.ContextKey == "" {
		msg := res.ErrorMessage
		if msg == "" {
			msg = "invalid credentials"
		}
		return fmt.Errorf("login failed: %s", msg)
	}

	m.contextKey = res.LoginData.ContextKey
	m.log.DEBUG.Printf("login ok: %s", res.LoginData.Name)
	return nil
}

// authHeaders returns request headers including the current context key
func (m *Melcloud) authHeaders() map[string]string {
	return map[string]string{
		"Content-Type":     request.JSONContent,
		"Accept":           request.JSONContent,
		"X-MitsContextKey": m.contextKey,
		"X-Requested-With": "XMLHttpRequest",
	}
}

// selectDevice fetches the device list and picks the requested device by id or name.
// If id is empty and exactly one device exists, it is returned.
func (m *Melcloud) selectDevice(id string) (melcloudListDevice, error) {
	return ensureEx("melcloud device", id, func() ([]melcloudListDevice, error) {
		var buildings []melcloudBuilding
		if err := m.apiGet("/User/ListDevices", &buildings); err != nil {
			return nil, err
		}

		var devices []melcloudListDevice
		for _, b := range buildings {
			devices = append(devices, b.Structure.Devices...)
			for _, a := range b.Structure.Areas {
				devices = append(devices, a.Devices...)
			}
			for _, f := range b.Structure.Floors {
				devices = append(devices, f.Devices...)
				for _, a := range f.Areas {
					devices = append(devices, a.Devices...)
				}
			}
		}
		return devices, nil
	}, func(d melcloudListDevice) (string, error) {
		if d.DeviceName != "" {
			return d.DeviceName, nil
		}
		return strconv.FormatInt(d.DeviceID, 10), nil
	})
}

// apiGet performs an authenticated GET, refreshing the token on auth failure
func (m *Melcloud) apiGet(path string, res any) error {
	return m.apiCall(http.MethodGet, path, nil, res)
}

// apiPost performs an authenticated POST, refreshing the token on auth failure
func (m *Melcloud) apiPost(path string, body any, res any) error {
	return m.apiCall(http.MethodPost, path, body, res)
}

func (m *Melcloud) apiCall(method, path string, body any, res any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	do := func() error {
		var reader = request.MarshalJSON(body)
		if body == nil {
			reader = nil
		}
		req, err := request.New(method, melcloudURI+path, reader, m.authHeaders())
		if err != nil {
			return err
		}
		return m.DoJSON(req, res)
	}

	err := do()
	if err == nil {
		return nil
	}

	// retry once on auth failure
	if se, ok := errors.AsType[*request.StatusError](err); ok && se.HasStatus(http.StatusUnauthorized, http.StatusForbidden) {
		if err := m.login(); err != nil {
			return err
		}
		return do()
	}

	return err
}

// fetchState retrieves the current device state as a generic map
func (m *Melcloud) fetchState() (melcloudDeviceState, error) {
	path := fmt.Sprintf("/Device/Get?id=%d&buildingID=%d", m.deviceID, m.buildingID)
	var state melcloudDeviceState
	if err := m.apiGet(path, &state); err != nil {
		return nil, err
	}
	return state, nil
}

// setState submits the modified state back to MELCloud
func (m *Melcloud) setState(state melcloudDeviceState) error {
	path := "/Device/SetAta"
	if m.deviceType == melcloudATW {
		path = "/Device/SetAtw"
	}
	state["HasPendingCommand"] = true
	return m.apiPost(path, state, &state)
}

// tempParams returns the temperature state key and effective flags for the device type
func (m *Melcloud) tempParams() (key string, powerFlag, tempFlag int64) {
	switch {
	case m.deviceType == melcloudATA:
		return "SetTemperature", melcloudFlagAtaPower, melcloudFlagAtaSetTemperature
	case m.useTank:
		return "SetTankWaterTemperature", melcloudFlagAtwPower, melcloudFlagAtwSetTankTemperature
	default:
		return "SetTemperatureZone1", melcloudFlagAtwPower, melcloudFlagAtwSetTemperatureZone1
	}
}

// setMode applies an SG Ready mode: Dim powers the device off, Normal/Boost
// power it on and set the corresponding temperature setpoint
func (m *Melcloud) setMode(mode int64) error {
	state, err := m.fetchState()
	if err != nil {
		return err
	}

	var setpoint float64
	switch mode {
	case Dim:
		state["Power"] = false
	case Normal:
		state["Power"] = true
		setpoint = m.normalTemp
	case Boost:
		state["Power"] = true
		setpoint = m.boostTemp
	default:
		return api.ErrNotAvailable
	}

	key, powerFlag, tempFlag := m.tempParams()

	flags := powerFlag
	if mode != Dim {
		state[key] = setpoint
		flags |= tempFlag
	}
	state["EffectiveFlags"] = flags

	return m.setState(state)
}

// getMode reports the current SG Ready mode inferred from the device state:
// powered off maps to Dim, otherwise the setpoint distinguishes Normal from Boost
func (m *Melcloud) getMode() (int64, error) {
	state, err := m.fetchState()
	if err != nil {
		return 0, err
	}

	if power, ok := state["Power"].(bool); ok && !power {
		return Dim, nil
	}

	key, _, _ := m.tempParams()
	if setpoint, ok := state[key].(float64); ok && setpoint >= m.boostTemp {
		return Boost, nil
	}
	return Normal, nil
}

// temperature returns the current room or tank temperature
func (m *Melcloud) temperature() (float64, error) {
	state, err := m.fetchState()
	if err != nil {
		return 0, err
	}

	keys := []string{"RoomTemperature", "RoomTemperatureZone1"}
	if m.deviceType == melcloudATW && m.useTank {
		keys = []string{"TankWaterTemperature", "RoomTemperatureZone1"}
	}

	for _, k := range keys {
		if v, ok := state[k]; ok {
			if f, ok := v.(float64); ok && f != 0 {
				return f, nil
			}
		}
	}
	return 0, api.ErrNotAvailable
}
