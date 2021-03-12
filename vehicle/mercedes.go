package vehicle

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

// Mercedes is an api.Vehicle implementation for Mercedes cars
type Mercedes struct {
	*embed
	oc    *oauth2.Config
	token *oauth2.Token
}

func init() {
	registry.Add("mercedes", NewMercedesFromConfig)
}

func state() string {
	var b [9]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

// openURL opens the specified URL in the default browser of the user.
func openURL(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// NewMercedesFromConfig creates a new Mercedes vehicle
func NewMercedesFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title                  string
		Capacity               int64
		ClientID, ClientSecret string
		User, Password         string
		Tokens                 Tokens
		VIN                    string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" && cc.Tokens.Access == "" {
		return nil, errors.New("missing credentials")
	}

	config := &oauth2.Config{
		ClientID:     cc.ClientID,
		ClientSecret: cc.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  "https://id.mercedes-benz.com/as/token.oauth2",
			AuthURL:   "https://id.mercedes-benz.com/as/authorization.oauth2",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		// Scopes: []string{"mb:vehicle:status:general", "mb:user:pool:reader", "offline_access"},
		Scopes: []string{"offline_access"},
	}

	v := &Mercedes{
		embed: &embed{cc.Title, cc.Capacity},
		oc:    config,
	}

	log := util.NewLogger("mercds")
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewHelper(log).Client)

	state := state()
	url := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "login consent"))
	fmt.Println(url)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	http.HandleFunc("/", v.redirectHandler(ctx, wg, state))
	go func() {
		wg.Done()
		http.ListenAndServe(":34972", nil)
	}()

	wg.Add(1)
	if err := openURL(url); err != nil {
		return v, err
	}

	go func() {
		time.Sleep(10 * time.Second)
		wg.Done()
	}()

	wg.Wait()
	// authenticated http client with logging injected to the Mercedes client

	// vehicles, err := client.Vehicles()
	// if err != nil {
	// 	return nil, err
	// }

	// if cc.VIN == "" && len(vehicles) == 1 {
	// 	v.vehicle = vehicles[0]
	// } else {
	// 	for _, vehicle := range vehicles {
	// 		if vehicle.Vin == strings.ToUpper(cc.VIN) {
	// 			v.vehicle = vehicle
	// 		}
	// 	}
	// }

	return v, nil
}

func (v *Mercedes) redirectHandler(ctx context.Context, wg *sync.WaitGroup, state string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()

		data, err := url.ParseQuery(r.URL.RawQuery)
		if error, ok := data["error"]; ok {
			fmt.Fprintf(w, "error: %s: %s\n", error, data["error_description"])
			return
		}

		states, ok := data["state"]
		if !ok || len(states) != 1 || states[0] != state {
			fmt.Fprintln(w, "invalid response:", data)
			return
		}

		codes, ok := data["code"]
		if !ok || len(codes) != 1 {
			fmt.Fprintln(w, "invalid response:", data)
			return
		}

		if v.token, err = v.oc.Exchange(ctx, codes[0]); err != nil {
			fmt.Fprintln(w, "token error:", err)
			return
		}

		fmt.Fprintln(w, "Folgende Fahrzeugkonfiguration kann in die evcc.yaml Konfigurationsdatei übernommen werden")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  tokens:")
		fmt.Fprintln(w, "    access:", v.token.AccessToken)
		fmt.Fprintln(w, "    refresh:", v.token.RefreshToken)
	}
}

// chargeState implements the api.Vehicle interface
func (v *Mercedes) chargeState() (float64, error) {
	// state, err := v.vehicle.ChargeState()
	// if err != nil {
	// 	return 0, err
	// }
	// return float64(state.BatteryLevel), nil
	return 0, nil
}

// SoC implements the api.Vehicle interface
func (v *Mercedes) SoC() (float64, error) {
	// return v.chargeStateG()
	return 0, nil
}
