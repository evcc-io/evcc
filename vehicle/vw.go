package vehicle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

const (
	vwBaseUrl     = "https://msg.volkswagen.de/fs-car"
	vwTokenUrl    = vwBaseUrl + "/core/auth/v1/VW/DE/token"
	vwVehiclesUrl = vwBaseUrl + "/usermanagement/users/v1/VW/DE/vehicles"
	vwBatteryUrl  = vwBaseUrl + "/bs/batterycharge/v1/VW/DE/vehicles/%s/charger"
)

// VW is an api.Vehicle implementation for VW cars
type VW struct {
	*embed
	*util.HTTPHelper
	user, password string
	token          string
	chargeStateG   provider.FloatGetter
}

// NewVWFromConfig creates a new vehicle
func NewVWFromConfig(log *util.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
	util.DecodeOther(log, other, &cc)

	// if err := session.Connect(cc.User, cc.Password); err != nil {
	// 	log.FATAL.Fatalf("cannot create VW: %v", err)
	// }

	v := &VW{
		embed:      &embed{cc.Title, cc.Capacity},
		HTTPHelper: util.NewHTTPHelper(util.NewLogger("vw")),
		user:       cc.User,
		password:   cc.Password,
	}

	v.login()
	// v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v
}

func (v *VW) headers(header *http.Header) {
	for k, v := range map[string]string{
		"Accept":        "application/json",
		"X-App-Name":    "eRemote",
		"X-App-Version": "1.0.0",
		"User-Agent":    "okhttp/2.3.0",
	} {
		header.Set(k, v)
	}

	if v.token != "" {
		header.Set("Authorization", "AudiAuth 1 "+v.token)
	}
}

func (v *VW) login() error {
	data := url.Values{
		"grant_type": []string{"password"},
		"username":   []string{v.user},
		"password":   []string{v.password},
	}

	req, err := http.NewRequest(http.MethodPost, vwTokenUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	v.headers(&req.Header)

	b, _ := httputil.DumpRequest(req, true)
	println(string(b))

	res := map[string]interface{}{}
	s, err := v.RequestJSON(req, res)
	println(s)
	if err != nil {
		return err
	}

	res1 := struct{ AccessToken string }{}
	if err := json.Unmarshal(s, &res1); err != nil {
		return err
	}

	fmt.Println(res)

	// 	expires, err := strconv.Atoi(query.Get("expires_in"))
	// if err != nil || token == "" || expires == 0 {
	// 	return errors.New("could not obtain token")
	// }

	v.token = res1.AccessToken
	// v.tokenValid = time.Now().Add(time.Duration(expires) * time.Second)

	req, err = http.NewRequest(http.MethodGet, vwVehiclesUrl, nil)
	if err != nil {
		return err
	}
	v.headers(&req.Header)

	b, _ = httputil.DumpRequest(req, true)
	println(string(b))

	res2 := map[string]interface{}{}
	b, err = v.RequestJSON(req, &res2)
	if err != nil {
		return err
	}
	fmt.Printf("%+v", string(b))
	// print "Retrieving verhicle"
	// VIN = responseData.get("userVehicles").get("vehicle")[0]

	req, err = http.NewRequest(http.MethodGet, fmt.Sprintf(vwBatteryUrl, "VIN"), nil)
	if err != nil {
		return err
	}
	v.headers(&req.Header)

	b, _ = httputil.DumpRequest(req, true)
	println(string(b))

	res3 := map[string]interface{}{}
	b, err = v.RequestJSON(req, &res3)
	if err != nil {
		return err
	}
	fmt.Printf("%+v", string(b))

	// #print "Charger request: " + r.content
	// // responseData.get("charger").
	// chargingMode = get("status").get("chargingStatusData").get("chargingMode").get("content")
	// chargingReason = get("status").get("chargingStatusData").get("chargingReason").get("content")
	// externalPowerSupplyState = get("status").get("chargingStatusData").get("externalPowerSupplyState").get("content")
	// energyFlow = get("status").get("chargingStatusData").get("energyFlow").get("content")
	// chargingState = get("status").get("chargingStatusData").get("chargingState").get("content")
	// stateOfCharge = get("status").get("batteryStatusData").get("stateOfCharge").get("content")
	// remainingChargingTime = get("status").get("batteryStatusData").get("remainingChargingTime").get("content")
	// remainingChargingTimeTargetSOC = get("status").get("batteryStatusData").get("remainingChargingTimeTargetSOC").get("content")
	// primaryEngineRange = get("status").get("cruisingRangeStatusData").get("primaryEngineRange").get("content")

	return nil
}

// chargeState implements the Vehicle.ChargeState interface
func (v *VW) chargeState() (float64, error) {
	// bs, err := v.session.BatteryStatus()
	// return float64(bs.StateOfCharge), err
	return 0, nil
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *VW) ChargeState() (float64, error) {
	return v.chargeStateG()
}
