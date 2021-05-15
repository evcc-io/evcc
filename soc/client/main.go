package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle"
	"github.com/andig/evcc/util"
)

func usage() {
	fmt.Print(`
soc

Usage:
  soc brand [--log level] [--param value [...]]
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
		start := time.Now()
		for {
			if time.Since(start) > time.Minute {
				log.Fatal(api.ErrTimeout)
			}

			soc, err := v.SoC()
			if err != nil {
				if errors.As(err, &api.ErrMustRetry) {
					time.Sleep(5 * time.Second)
					continue
				}

				log.Fatal(err)
			}

			fmt.Println(int(math.Round(soc)))
			break
		}

	default:
		log.Fatal("invalid action:", action)
	}
}
