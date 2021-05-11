package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle"
	"github.com/andig/evcc/util"
)

func usage() {
	fmt.Print(`
soc

Usage:
  soc vehicle [--log level] [--param value [...]]
`)
}

func main() {
	if len(os.Args) < 3 {
		usage()
		log.Fatal("not enough arguments")
	}

	params := make(map[string]interface{})
	params["brand"] = strings.ToLower(os.Args[1])

	action := "soc"

	var key string
	for _, arg := range os.Args[2:] {
		switch key {
		case "":
			key = strings.ToLower(strings.TrimLeft(arg, "-"))
		case "log":
			util.LogLevel(arg, nil)
			key = ""
		case "action":
			action = arg
			key = ""
		default:
			params[key] = arg
			key = ""
		}
	}

	if key != "" {
		usage()
		log.Fatal("unexpected number of parameters")
	}

	v, err := vehicle.NewCloudFromConfig(params)
	if err != nil {
		log.Fatal(err)
	}

	switch action {
	case "wakeup":
		vv, ok := v.(api.VehicleStartCharge)
		if !ok {
			log.Fatal("not supported:", action)
		}
		if err := vv.StartCharge(); err != nil {
			log.Fatal(err)
		}

	case "soc":
		soc, err := v.SoC()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(int(math.Round(soc)))

	default:
		log.Fatal("invalid action:", action)
	}
}
