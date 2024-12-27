package charger

// LICENSE

// Copyright (c) 2024 andig

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

	"github.com/WulfgarW/sensonet"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("vaillant", NewVaillantFromConfig)
}

// NewVaillantFromConfig creates an Vaillant configurable charger from generic config
func NewVaillantFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed           `mapstructure:",squash"`
		User, Password  string
		Realm           string
		HeatingZone     int
		HeatingSetpoint float32
		Phases          int
	}{
		embed: embed{
			Icon_:     "heatpump",
			Features_: []api.Feature{api.Heating, api.IntegratedDevice},
		},
		Realm:  sensonet.REALM_GERMANY,
		Phases: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("vaillant").Redact(cc.User, cc.Password)
	client := request.NewClient(log)
	clientCtx := context.WithValue(ctx, oauth2.HTTPClient, client)

	oc := sensonet.Oauth2ConfigForRealm(cc.Realm)
	token, err := oc.PasswordCredentialsToken(clientCtx, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	conn, err := sensonet.NewConnection(client, oc.TokenSource(clientCtx, token))
	if err != nil {
		return nil, err
	}

	homes, err := conn.GetHomes()
	if err != nil {
		return nil, err
	}

	systemID := homes[0].SystemID
	heatingPar := sensonet.HeatingParStruct{
		ZoneIndex:    cc.HeatingZone,
		VetoSetpoint: cc.HeatingSetpoint,
		VetoDuration: -1, // negative value means: use default
	}
	hotwaterPar := sensonet.HotwaterParStruct{
		Index: -1, // negative value means: use default
	}

	set := func(mode int64) error {
		switch mode {
		case Normal:
			_, err := conn.StopStrategybased(systemID, &heatingPar, &hotwaterPar)
			return err
		case Boost:
			strategy := sensonet.STRATEGY_HOTWATER
			if cc.HeatingSetpoint != 0 {
				strategy = sensonet.STRATEGY_HOTWATER_THEN_HEATING
			}
			_, err := conn.StartStrategybased(systemID, strategy, &heatingPar, &hotwaterPar)
			return err
		default:
			return api.ErrNotAvailable
		}
	}

	return NewSgReady(ctx, &cc.embed, set, nil, nil, cc.Phases)
}
