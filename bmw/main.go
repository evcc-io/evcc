package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"slices"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/bmw/cardata"
	_ "github.com/joho/godotenv/autoload"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"golang.org/x/oauth2"
)

const (
	apiUrl   = "https://api-cardata.bmwgroup.com"
	fileName = ".bmw-token.json"
)

var Config = &oauth2.Config{
	ClientID: os.Getenv("CLIENT_ID"),
	Scopes:   []string{"authenticate_user", "openid", "cardata:api:read", "cardata:streaming:read"},
	Endpoint: oauth2.Endpoint{
		DeviceAuthURL: "https://customer.bmwgroup.com/gcdm/oauth/device/code",
		TokenURL:      "https://customer.bmwgroup.com/gcdm/oauth/token",
		AuthStyle:     oauth2.AuthStyleInParams,
	},
}

func generateToken(ctx context.Context) (*oauth2.Token, error) {
	cv := oauth2.GenerateVerifier()

	da, err := Config.DeviceAuth(ctx,
		oauth2.S256ChallengeOption(cv),
	)
	if err != nil {
		return nil, err
	}

	fmt.Println("open and confirm:", da.VerificationURIComplete)
	bufio.NewReader(os.Stdin).ReadLine()

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	token, err := Config.DeviceAccessToken(ctx, da,
		oauth2.VerifierOption(cv),
	)
	if err != nil {
		return nil, err
	}

	return token, storeToken(token)
}

func storeToken(token *oauth2.Token) error {
	cdToken := &cardata.Token{
		Token:   token,
		IdToken: tokenExtra(token, "id_token"),
		Gcid:    tokenExtra(token, "gcid"),
	}

	b, _ := json.Marshal(cdToken)
	return os.WriteFile(fileName, b, 0o644)
}

func loadToken() (*oauth2.Token, error) {
	var token *cardata.Token

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(f).Decode(&token); err != nil {
		return nil, err
	}

	return token.TokenEx(), nil
}

func tokenExtra(t *oauth2.Token, key string) string {
	if v := t.Extra(key); v != nil {
		return v.(string)
	}
	return ""
}

func getVehicleMappings(client *request.Helper) ([]cardata.VehicleMapping, error) {
	var res []cardata.VehicleMapping
	err := client.GetJSON(apiUrl+"/customers/vehicles/mappings", &res)

	return lo.Filter(res, func(v cardata.VehicleMapping, _ int) bool {
		return v.MappingType == "PRIMARY"
	}), err
}

func getContainers(client *request.Helper) ([]cardata.Container, error) {
	var res struct {
		Containers []cardata.Container
	}
	if err := client.GetJSON(apiUrl+"/customers/containers", &res); err != nil {
		return nil, err
	}
	return lo.Filter(res.Containers, func(c cardata.Container, _ int) bool {
		return c.Name == "evcc.io"
	}), nil
}

func callApi(client *request.Helper, deleteContainer *bool) error {
	mappings, err := getVehicleMappings(client)
	if err != nil {
		return err
	}
	if len(mappings) == 0 {
		return errors.New("could not find primary vehicle mapping")
	}
	vin := mappings[0].Vin

	containers, err := getContainers(client)
	if err != nil {
		return err
	}

	if *deleteContainer && len(containers) == 1 {
		req, _ := request.New(http.MethodDelete, apiUrl+"/customers/containers/"+containers[0].ContainerId, nil)

		var res any
		if err := client.DoJSON(req, &res); err != nil {
			return err
		}

		containers = nil
	}

	if len(containers) == 0 {
		data := cardata.CreateContainer{
			Name:    "evcc.io",
			Purpose: "evcc.io",
			TechnicalDescriptors: []string{
				// https://mybmwweb-utilities.api.bmw/de-de/utilities/bmw/api/cd/catalogue/file
				"vehicle.body.chargingPort.status",
				"vehicle.cabin.hvac.preconditioning.status.comfortState",
				"vehicle.drivetrain.batteryManagement.header",
				"vehicle.drivetrain.electricEngine.charging.hvStatus",
				"vehicle.drivetrain.electricEngine.charging.level",
				"vehicle.drivetrain.electricEngine.charging.timeToFullyCharged",
				"vehicle.drivetrain.electricEngine.kombiRemainingElectricRange",
				"vehicle.powertrain.electric.battery.stateOfCharge.target",
				"vehicle.vehicle.travelledDistance",
			},
		}
		req, _ := request.New(http.MethodPost, apiUrl+"/customers/containers", request.MarshalJSON(data))

		var res any
		if err := client.DoJSON(req, &res); err != nil {
			return err
		}

		if containers, err = getContainers(client); err != nil {
			return err
		}
	}

	if len(containers) > 0 {
		var res cardata.TelematicData
		uri := fmt.Sprintf(apiUrl+"/customers/vehicles/%s/telematicData?containerId=%s", vin, containers[0].ContainerId)
		if err := client.GetJSON(uri, &res); err != nil {
			return err
		}

		for _, k := range slices.Sorted(maps.Keys(res.TelematicData)) {
			v := res.TelematicData[k]
			val := (any)(v.Value)
			if f, err := cast.ToFloat64E(v.Value); err == nil {
				val = f
			}
			fmt.Println(k, v.Timestamp, val, v.Unit)
		}
	}

	return nil
}

func runMqtt(token *oauth2.Token) error {
	gcid := tokenExtra(token, "gcid")
	idToken := tokenExtra(token, "id_token")

	o := mqtt.NewClientOptions().
		AddBroker("tls://customer.streaming-cardata.bmwgroup.com:9000").
		SetAutoReconnect(true).
		SetUsername(gcid).
		SetPassword(idToken)

	paho := mqtt.NewClient(o)

	timeout := 30 * time.Second
	if t := paho.Connect(); !t.WaitTimeout(timeout) {
		log.Fatal("connect timeout")
	} else if err := t.Error(); err != nil {
		log.Fatal("connect:", err)
	}
	defer paho.Disconnect(0)

	topic := fmt.Sprintf("%s/%s", gcid, os.Getenv("VIN"))

	fmt.Println("gcid:", gcid)
	fmt.Println("id_token:", idToken)
	fmt.Println("topic:", topic)

	if t := paho.Subscribe(topic, 0, func(c mqtt.Client, m mqtt.Message) {
		var msg cardata.StreamingMessage
		if err := json.Unmarshal(m.Payload(), &msg); err == nil {
			fmt.Printf("%+v", msg)
		} else {
			fmt.Println(m.Topic(), string(m.Payload()), err)
		}
	}); !t.WaitTimeout(timeout) {
		log.Fatal("subcribe timeout")
	} else if err := t.Error(); err != nil {
		log.Fatal("subscribe:", err)
	}

	until := time.Until(token.Expiry)
	fmt.Println("until:", until)
	time.Sleep(until)

	return nil
}

func main() {
	apiCall := flag.Bool("api", false, "include api")
	mqttCall := flag.Bool("mqtt", false, "include mqtt")
	deleteContainer := flag.Bool("delete", false, "delete container")
	oauthRefresh := flag.Bool("refresh", false, "refresh token")
	flag.Parse()

	util.LogLevel("trace", nil)
	request.LogHeaders = true

	client := request.NewHelper(util.NewLogger("foo"))
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client.Client)

	token, err := loadToken()
	if err != nil {
		if token, err = generateToken(ctx); err != nil {
			log.Fatal(err)
		}
	}

	if *oauthRefresh {
		token.Expiry = time.Now()
	}

	ts := &cardata.PersistingTokenSource{
		TokenSource: Config.TokenSource(ctx, token),
		Persist:     storeToken,
	}

	if *apiCall {
		apiClient := client
		apiClient.Transport = &oauth2.Transport{
			Source: ts,
			Base: &transport.Decorator{
				Decorator: transport.DecorateHeaders(map[string]string{
					"x-version": "v1",
				}),
				Base: client.Transport,
			},
		}

		if err := callApi(apiClient, deleteContainer); err != nil {
			log.Fatal(err)
		}
	}

	if *mqttCall {
		mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
		mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
		// mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
		// mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)

		fmt.Println("connecting to mqtt")

		for {
			token, err := ts.Token()
			if err != nil {
				log.Fatal(err)
			}

			if err := runMqtt(token); err != nil {
				log.Fatal(err)
			}
		}
	}
}
