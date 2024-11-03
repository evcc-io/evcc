package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/meter/zendure"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	util.LogLevel("trace", nil)

	res, err := zendure.MqttCredentials(os.Getenv("ZENDURE_ACCOUNT"), os.Getenv("ZENDURE_SERIAL"))
	if err != nil {
		panic(err)
	}

	fmt.Println(res)

	client, err := mqtt.NewClient(util.NewLogger("mqtt"), net.JoinHostPort(res.Data.MqttUrl, strconv.Itoa(res.Data.Port)), res.Data.AppKey, res.Data.Secret, "", 0, false, "", "", "")
	if err != nil {
		panic(err)
	}

	if err := client.Listen("#", func(data string) {
		fmt.Println(data)
	}); err != nil {
		panic(err)
	}

	time.Sleep(time.Hour)
}
