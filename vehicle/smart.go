package vehicle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/publicsuffix"
)

// Smart is an api.Vehicle implementation for Smart cars
type Smart struct {
	*embed
	*request.Helper
}

func init() {
	registry.Add("smart", NewSmartFromConfig)
}

// NewSmartFromConfig creates a new vehicle
func NewSmartFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Expiry         time.Duration
		Cache          time.Duration
	}{
		Expiry: expiry,
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("smart").Redact(cc.User, cc.Password, cc.VIN)

	v := &Smart{
		embed:  &cc.embed,
		Helper: request.NewHelper(log),
	}

	// method: "post",
	// url: "https://id.mercedes-benz.com/ciam/auth/login/user",
	// headers: {
	// 	"Content-Type": "application/json",
	// 	Accept: "application/json, text/plain, */*",
	// 	"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 12_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
	// 	Referer: "https://id.mercedes-benz.com/ciam/auth/login",
	// 	"Accept-Language": "de-de",
	// },
	// jar: this.cookieJar,
	// withCredentials: true,
	// data: JSON.stringify({
	// 	username: this.config.username,
	// }),
	// })
	// .then((res) => {
	// 	this.log.debug(JSON.stringify(res.data));
	// 	this.session = res.data;
	// 	this.setState("info.connection", true, true);
	// })
	// .catch((error) => {
	// 	this.log.error(error);
	// 	if (error.response) {
	// 		this.log.error(JSON.stringify(error.response.data));
	// 	}
	// });

	data := struct {
		Username   string `json:"username"`
		Password   string `json:"password,omitempty"`
		RememberMe bool   `json:"rememberMe,omitempty"`
	}{
		Username: cc.User,
	}

	v.Client.Jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	var CodeVerifier, _ = cv.CreateCodeVerifier()
	codeChallenge := CodeVerifier.CodeChallengeS256()

	// const [code_verifier, codeChallenge] = this.getCodeChallenge();
	// const resume = await this.requestClient({
	// method: "get",
	// url:
	// 	"https://id.mercedes-benz.com/as/authorization.oauth2?client_id=70d89501-938c-4bec-82d0-6abb550b0825&response_type=code&scope=openid+profile+email+phone+ciam-uid+offline_access&redirect_uri=https://oneapp.microservice.smart.com&code_challenge=" +
	// 	codeChallenge +
	// 	"&code_challenge_method=S256",
	// headers: {
	// 	Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	// 	"Accept-Language": "de-de",
	// 	"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 12_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
	// },
	// jar: this.cookieJar,
	// withCredentials: true,
	// })
	// .then((res) => {
	// 	this.log.debug(JSON.stringify(res.data));
	// 	return qs.parse(res.request.path.split("?")[1]).resume;
	// })
	// .catch((error) => {
	// 	this.log.error(error);
	// 	if (error.response) {
	// 		this.log.error(JSON.stringify(error.response.data));
	// 	}
	// });

	dataTokenAuth := url.Values{
		"redirect_uri":          []string{redirectURI},
		"client_id":             []string{actualClientID},
		"response_type":         []string{"code"},
		"state":                 []string{"uvobn7XJs1"},
		"scope":                 []string{"openid"},
		"access_type":           []string{"offline"},
		"country":               []string{"de"},
		"locale":                []string{"de_DE"},
		"code_challenge":        []string{codeChallenge},
		"code_challenge_method": []string{"S256"},
	}

	req, err := http.NewRequest(http.MethodGet, "https://login.porsche.com/as/authorization.oauth2", nil)
	if err != nil {
		return pr, err
	}

	// req, err := request.New(http.MethodPost, "https://id.mercedes-benz.com/ciam/auth/login/user", request.MarshalJSON(data), request.JSONEncoding)
	req, err := request.New(http.MethodPost, "https://id.mercedes-benz.com/ciam/auth/login/user", request.MarshalJSON(data), map[string]string{
		"Content-Type":    request.JSONContent,
		"Accept":          request.JSONContent,
		"Referer":         "https://id.mercedes-benz.com/ciam/auth/login",
		"Accept-Language": "de-de",
	})
	if err != nil {
		return nil, err
	}

	var res struct {
		Errors []json.RawMessage
	}
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	//     const token = await this.requestClient({
	//     method: "post",
	//     url: "https://id.mercedes-benz.com/ciam/auth/login/pass",
	//     headers: {
	//         "Content-Type": "application/json",
	//         Accept: "application/json, text/plain, */*",
	//         "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 12_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
	//         Referer: "https://id.mercedes-benz.com/ciam/auth/login",
	//         "Accept-Language": "de-de",
	//     },
	//     jar: this.cookieJar,
	//     withCredentials: true,
	//     data: JSON.stringify({
	//         username: this.config.username,
	//         password: this.config.password,
	//         rememberMe: true,
	//     }),
	// })

	data.Password = cc.Password
	data.RememberMe = true

	// req, err = request.New(http.MethodPost, "https://id.mercedes-benz.com/ciam/auth/login/pass", request.MarshalJSON(data), request.JSONEncoding)
	req, err = request.New(http.MethodPost, "https://id.mercedes-benz.com/ciam/auth/login/pass", request.MarshalJSON(data), map[string]string{
		"Content-Type":    request.JSONContent,
		"Accept":          request.JSONContent,
		"Referer":         "https://id.mercedes-benz.com/ciam/auth/login",
		"Accept-Language": "de-de",
	})
	if err != nil {
		return nil, err
	}

	// var res interface{}
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	fmt.Printf("%+v\n", res)
	return v, nil
}

// SoC implements the api.Vehicle interface
func (v *Smart) SoC() (float64, error) {

	return 0, nil
}

// var _ api.ChargeState = (*Smart)(nil)

// // Status implements the api.ChargeState interface
// func (v *Smart) Status() (api.ChargeStatus, error) {
// 	status := api.StatusA // disconnected
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Smart.ChargeState); err == nil && ok {
// 		if res.ChargingState == "Stopped" || res.ChargingState == "NoPower" || res.ChargingState == "Complete" {
// 			status = api.StatusB
// 		}
// 		if res.ChargingState == "Charging" {
// 			status = api.StatusC
// 		}
// 	}

// 	return status, err
// }

// var _ api.ChargeRater = (*Smart)(nil)

// // ChargedEnergy implements the api.ChargeRater interface
// func (v *Smart) ChargedEnergy() (float64, error) {
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Smart.ChargeState); err == nil && ok {
// 		return res.ChargeEnergyAdded, nil
// 	}

// 	return 0, err
// }

// const kmPerMile = 1.609344

// var _ api.VehicleRange = (*Smart)(nil)

// // Range implements the api.VehicleRange interface
// func (v *Smart) Range() (int64, error) {
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Smart.ChargeState); err == nil && ok {
// 		// miles to km
// 		return int64(kmPerMile * res.BatteryRange), nil
// 	}

// 	return 0, err
// }

// var _ api.VehicleOdometer = (*Smart)(nil)

// // Odometer implements the api.VehicleOdometer interface
// func (v *Smart) Odometer() (float64, error) {
// 	res, err := v.vehicleStateG()

// 	if res, ok := res.(*Smart.VehicleState); err == nil && ok {
// 		// miles to km
// 		return kmPerMile * res.Odometer, nil
// 	}

// 	return 0, err
// }

// var _ api.VehicleFinishTimer = (*Smart)(nil)

// // FinishTime implements the api.VehicleFinishTimer interface
// func (v *Smart) FinishTime() (time.Time, error) {
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Smart.ChargeState); err == nil && ok {
// 		t := time.Now()
// 		return t.Add(time.Duration(res.MinutesToFullCharge) * time.Minute), err
// 	}

// 	return time.Time{}, err
// }

// // TODO api.Climater implementation has been removed as it drains battery. Re-check at a later time.

// var _ api.VehiclePosition = (*Smart)(nil)

// // Position implements the api.VehiclePosition interface
// func (v *Smart) Position() (float64, float64, error) {
// 	res, err := v.driveStateG()
// 	if res, ok := res.(*Smart.DriveState); err == nil && ok {
// 		return res.Latitude, res.Longitude, nil
// 	}

// 	return 0, 0, err
// }

// var _ api.VehicleStartCharge = (*Smart)(nil)

// // StartCharge implements the api.VehicleStartCharge interface
// func (v *Smart) StartCharge() error {
// 	err := v.vehicle.StartCharging()

// 	if err != nil && err.Error() == "408 Request Timeout" {
// 		if _, err := v.vehicle.Wakeup(); err != nil {
// 			return err
// 		}

// 		timer := time.NewTimer(90 * time.Second)

// 		for {
// 			select {
// 			case <-timer.C:
// 				return api.ErrTimeout
// 			default:
// 				time.Sleep(2 * time.Second)
// 				if err := v.vehicle.StartCharging(); err == nil || err.Error() != "408 Request Timeout" {
// 					return err
// 				}
// 			}
// 		}
// 	}

// 	return err
// }

// var _ api.VehicleStopCharge = (*Smart)(nil)

// // StopCharge implements the api.VehicleStopCharge interface
// func (v *Smart) StopCharge() error {
// 	err := v.vehicle.StopCharging()

// 	// ignore sleeping vehicle
// 	if err != nil && err.Error() == "408 Request Timeout" {
// 		err = nil
// 	}

// 	return err
// }
