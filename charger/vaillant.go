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
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/WulfgarW/sensonet"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("vaillant", NewVaillantFromConfig)
}

type Vaillant struct {
	*SgReady
	log      *util.Logger
	conn     *sensonet.Connection
	systemId string
}

//go:generate decorate -f decorateVaillant -b *Vaillant -r api.Charger -t "api.Battery,Soc,func() (float64, error)"

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

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("vaillant").Redact(cc.User, cc.Password)
	client := request.NewClient(log)
	clientCtx := context.WithValue(ctx, oauth2.HTTPClient, client)

	oc := sensonet.Oauth2ConfigForRealm(cc.Realm)
	token, err := oc.PasswordCredentialsToken(clientCtx, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	conn, err := sensonet.NewConnection(oc.TokenSource(clientCtx, token), sensonet.WithHttpClient(client), sensonet.WithLogger(log.TRACE))
	if err != nil {
		return nil, err
	}

	homes, err := conn.GetHomes()
	if err != nil {
		return nil, err
	}

	systemId := homes[0].SystemID

	system, err := conn.GetSystem(systemId)
	if err != nil {
		return nil, err
	}

	var strategy int
	hotWater := len(system.State.Dhw)+len(system.State.DomesticHotWater) > 0

	switch {
	case hotWater && cc.HeatingSetpoint != 0:
		strategy = sensonet.STRATEGY_HOTWATER_THEN_HEATING
	case hotWater:
		strategy = sensonet.STRATEGY_HOTWATER
	case cc.HeatingSetpoint != 0:
		strategy = sensonet.STRATEGY_HEATING
	default:
		return nil, errors.New("could not determine boost strategy, need either hot water or heating zone and setpoint")
	}

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
			_, err := conn.StopStrategybased(systemId, &heatingPar, &hotwaterPar)
			return err
		case Boost:
			_, err := conn.StartStrategybased(systemId, strategy, &heatingPar, &hotwaterPar)
			return err
		default:
			return api.ErrNotAvailable
		}
	}

	sgr, err := NewSgReady(ctx, &cc.embed, set, nil, nil, cc.Phases)
	if err != nil {
		return nil, err
	}

	res := &Vaillant{
		log:      log,
		conn:     conn,
		systemId: systemId,
		SgReady:  sgr,
	}

	var temp func() (float64, error)
	if hotWater {
		temp = func() (float64, error) {
			system, err := conn.GetSystem(systemId)
			if err != nil {
				return 0, err
			}

			switch {
			case len(system.State.Dhw) > 0:
				return system.State.Dhw[0].CurrentDhwTemperature, nil
			case len(system.State.DomesticHotWater) > 0:
				return system.State.DomesticHotWater[0].CurrentDomesticHotWaterTemperature, nil
			default:
				return 0, api.ErrNotAvailable
			}
		}
	}

	return decorateVaillant(res, temp), nil
}

func (v *Vaillant) print(chapter int, prefix string, zz ...any) {
	var i int
	for _, z := range zz {
		rt := reflect.TypeOf(z)
		rv := reflect.ValueOf(z)

		if rt.Kind() == reflect.Slice && rv.Len() == 0 {
			continue
		}
		i++

		typ := strings.TrimPrefix(strings.TrimPrefix(fmt.Sprintf("%T", z), "[]sensonet."), prefix)

		fmt.Println()
		fmt.Printf("%d.%d. %s\n", chapter, i+1, typ)

		if rt.Kind() == reflect.Slice {
			for j := 0; j < rv.Len(); j++ {
				fmt.Printf("%d.%d.%d. %s %d\n", chapter, i+1, j+1, typ, j)
				fmt.Printf("%+v\n", rv.Index(j))
			}
		} else {
			fmt.Printf("%+v\n", z)
		}
	}
}

func (v *Vaillant) Diagnose() {
	sys, err := v.conn.GetSystem(v.systemId)
	if err != nil {
		v.log.ERROR.Println(err)
		return
	}

	fmt.Println()
	fmt.Println("1. State")
	fmt.Println()
	fmt.Println("1.1. System")
	fmt.Printf("%+v\n", sys.State.System)
	v.print(1, "State", sys.State.Zones, sys.State.Circuits, sys.State.Dhw, sys.State.DomesticHotWater)

	fmt.Println()
	fmt.Println("2. Properties")
	fmt.Println()
	fmt.Println("2.1. System")
	fmt.Printf("%+v\n", sys.Properties.System)
	v.print(2, "Properties", sys.Properties.Zones, sys.Properties.Circuits, sys.Properties.Dhw, sys.Properties.DomesticHotWater)

	fmt.Println()
	fmt.Println("3. Configuration")
	fmt.Println()
	fmt.Println("3.1. System")
	fmt.Printf("%+v\n", sys.Configuration.System)
	v.print(3, "Configuration", sys.Configuration.Zones, sys.Configuration.Circuits, sys.Configuration.Dhw, sys.Configuration.DomesticHotWater)
}
