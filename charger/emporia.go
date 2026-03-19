package charger

// LICENSE

// Copyright (c) 2025 evcc contributors

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

// https://github.com/magico13/PyEmVue/blob/master/api_docs.md

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/emporia"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// Emporia charger implementation
type Emporia struct {
	*request.Helper
	deviceGid int64
	statusG   util.Cacheable[emporia.DevicesStatus]
}

func init() {
	registry.AddCtx("emporia", NewEmporiaFromConfig)
}

// NewEmporiaFromConfig creates an Emporia charger from generic config
func NewEmporiaFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		User      string
		Password  string
		DeviceGid int64
		Cache     time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewEmporia(ctx, cc.User, cc.Password, cc.DeviceGid, cc.Cache)
}

// NewEmporia creates an Emporia charger
func NewEmporia(ctx context.Context, user, password string, deviceGid int64, cache time.Duration) (*Emporia, error) {
	log := util.NewLogger("emporia").Redact(user, password)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	identity, err := emporia.NewIdentity(user, password)
	if err != nil {
		return nil, err
	}

	// Initial token (authenticates on creation)
	tok, err := identity.Login()
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	ts := oauth.RefreshTokenSource(tok, identity.Refresh)

	wb := &Emporia{
		Helper:    request.NewHelper(log),
		deviceGid: deviceGid,
	}

	// Use custom transport to add Emporia's authtoken header (not standard Bearer)
	wb.Client.Transport = &emporiaTransport{
		ts:   ts,
		base: wb.Client.Transport,
	}

	wb.statusG = util.ResettableCached(func() (emporia.DevicesStatus, error) {
		var res emporia.DevicesStatus
		err := wb.GetJSON(emporia.DevicesStatusEndpoint, &res)
		return res, err
	}, cache)

	// Auto-detect deviceGid if not provided
	if wb.deviceGid == 0 {
		status, err := wb.statusG.Get()
		if err != nil {
			return nil, fmt.Errorf("get devices status: %w", err)
		}
		if len(status.EvChargers) == 0 {
			return nil, fmt.Errorf("no EV chargers found on account")
		}
		if len(status.EvChargers) > 1 {
			return nil, fmt.Errorf("multiple EV chargers found, please specify deviceGid")
		}
		wb.deviceGid = status.EvChargers[0].DeviceGid
	}

	return wb, nil
}

// chargerStatus returns the current status of the Emporia charger
func (wb *Emporia) chargerStatus() (emporia.ChargerDevice, error) {
	status, err := wb.statusG.Get()
	if err != nil {
		return emporia.ChargerDevice{}, err
	}

	for _, c := range status.EvChargers {
		if c.DeviceGid == wb.deviceGid {
			return c, nil
		}
	}

	return emporia.ChargerDevice{}, fmt.Errorf("charger with deviceGid %d not found", wb.deviceGid)
}

// Status implements the api.Charger interface
func (wb *Emporia) Status() (api.ChargeStatus, error) {
	c, err := wb.chargerStatus()
	if err != nil {
		return api.StatusNone, err
	}

	if c.Status == "Charging" {
		return api.StatusC, nil
	}

	// Use icon field to determine if EV is connected
	if strings.HasPrefix(c.Icon, "Car") {
		return api.StatusB, nil
	}

	return api.StatusA, nil
}

// Enabled implements the api.Charger interface
func (wb *Emporia) Enabled() (bool, error) {
	c, err := wb.chargerStatus()
	if err != nil {
		return false, err
	}
	return c.ChargerOn, nil
}

// Enable implements the api.Charger interface
func (wb *Emporia) Enable(enable bool) error {
	c, err := wb.chargerStatus()
	if err != nil {
		return err
	}

	update := emporia.ChargerUpdate{
		DeviceGid:       c.DeviceGid,
		LoadGid:         c.LoadGid,
		ChargerOn:       enable,
		ChargingRate:    c.ChargingRate,
		MaxChargingRate: c.MaxChargingRate,
	}

	req, err := request.New(http.MethodPut, emporia.ChargerEndpoint, request.MarshalJSON(update), request.JSONEncoding)
	if err != nil {
		return err
	}

	var res emporia.ChargerDevice
	if err := wb.DoJSON(req, &res); err != nil {
		return err
	}

	wb.statusG.Reset()
	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Emporia) MaxCurrent(current int64) error {
	c, err := wb.chargerStatus()
	if err != nil {
		return err
	}

	update := emporia.ChargerUpdate{
		DeviceGid:       c.DeviceGid,
		LoadGid:         c.LoadGid,
		ChargerOn:       c.ChargerOn,
		ChargingRate:    int(current),
		MaxChargingRate: c.MaxChargingRate,
	}

	req, err := request.New(http.MethodPut, emporia.ChargerEndpoint, request.MarshalJSON(update), request.JSONEncoding)
	if err != nil {
		return err
	}

	var res emporia.ChargerDevice
	if err := wb.DoJSON(req, &res); err != nil {
		return err
	}

	wb.statusG.Reset()
	return nil
}

// emporiaTransport is an http.RoundTripper that adds the Emporia authtoken header
type emporiaTransport struct {
	ts   oauth2.TokenSource
	base http.RoundTripper
}

func (t *emporiaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.ts.Token()
	if err != nil {
		return nil, err
	}

	return (&transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"authtoken": tok.AccessToken,
		}),
		Base: t.base,
	}).RoundTrip(req)
}
