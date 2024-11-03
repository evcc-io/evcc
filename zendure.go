package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/meter/zendure"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
	_ "github.com/joho/godotenv/autoload"
)

// Topics

// acOutputPower
// acSwitch
// buzzerSwitch
// electricLevel
// gridInputPower
// outputHomePower
// outputLimit
// passMode
// remainOutTime
// socSet
// solarInputPower
// solarPower1
// solarPower2

// outputPackPower
// packData
// packInputPower
// packNum
// packState

func gridPower() (int64, error) {
	var state struct {
		Result struct {
			GridPower float64 `json:"gridPower"`
		}
	}

	resp, err := http.Get("http://nas.fritz.box:7070/api/state")
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&state)
	}

	return int64(state.Result.GridPower), err
}

func evcc(client *mqtt.Client, commands map[string]zendure.Command) {
	const (
		Margin       = 100
		MaxCharge    = -1000
		MaxDischarge = 500
	)

	var grid int64

	for range time.Tick(10 * time.Second) {
		res, err := gridPower()
		if err != nil {
			fmt.Println(err)
			continue
		}

		new := grid + res

		if cmd, ok := commands["outputHomePower"]; ok && new >= 0 {
			if new = max(0, min(new-Margin, MaxDischarge)); new != grid {
				fmt.Println("vvv")
				fmt.Println("set outputHomePower:", new)

				client.Publish(cmd.CommandTopic, false, strconv.FormatInt(new, 10))

				grid = new
			}
		} else if cmd, ok := commands["gridInputPower"]; ok && new < 0 {
			if new = min(0, max(new+Margin, MaxCharge)); new != grid {
				fmt.Println("^^^")
				fmt.Println("set gridInputPower:", new)

				client.Publish(cmd.CommandTopic, false, strconv.FormatInt(-new, 10))

				grid = new
			}
		}
	}
}

func main() {
	util.LogLevel("trace", nil)

	res, err := zendure.MqttCredentials(os.Getenv("ZENDURE_ACCOUNT"), os.Getenv("ZENDURE_SERIAL"))
	if err != nil {
		panic(err)
	}

	fmt.Println(res)

	client, err := mqtt.NewClient(
		util.NewLogger("mqtt"),
		net.JoinHostPort(res.Data.MqttUrl, strconv.Itoa(res.Data.Port)), res.Data.AppKey, res.Data.Secret,
		"", 0, false, "", "", "",
	)
	if err != nil {
		panic(err)
	}

	state := make(map[string]any)
	commands := make(map[string]zendure.Command)

	go evcc(client, commands)

	topic := res.Data.AppKey + "/#"

	if err := client.Listen(topic, func(data string) {
		fmt.Println(topic, ":", data)

		var cmd zendure.Command
		err := json.Unmarshal([]byte(data), &cmd)
		if err != nil {
			panic(err)
		}

		if full, ok := strings.CutSuffix(cmd.CommandTopic, "/set"); ok {
			segs := strings.Split(full, "/")
			key := segs[len(segs)-1]

			commands[key] = cmd

			if res, err := json.MarshalIndent(commands, "", "  "); err == nil {
				fmt.Println("===")
				fmt.Println(string(res))
			}

			return
		}

		var new map[string]any
		if err := json.Unmarshal([]byte(data), &new); err != nil {
			panic(err)
		}

		if err := mergo.Merge(&state, new, mergo.WithOverride); err != nil {
			panic(err)
		}

		if res, err := json.MarshalIndent(state, "", "  "); err == nil {
			fmt.Println("---")
			fmt.Println(string(res))
		}
	}); err != nil {
		panic(err)
	}

	time.Sleep(time.Hour)
}
