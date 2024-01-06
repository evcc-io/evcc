package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/vehicle"
)

func usage() {
	fmt.Print(`
soc

Usage:
  soc brand [--log level] [--param value [...]]
`)
}

// matchesError replaces errors.Is for errors returned from GRPC
func matchesError(err, match error) bool {
	return strings.Contains(err.Error(), match.Error())
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
		case "token":
			sponsor.Subject = arg // TODO placeholder
			sponsor.Token = arg
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
		vv, ok := v.(api.VehicleChargeController)
		if !ok {
			log.Fatal("not supported:", action)
		}
		if err := vv.StartCharge(); err != nil {
			log.Fatal(err)
		}

	case "soc":
		var soc float64
		var err error

		start := time.Now()
		for err = api.ErrMustRetry; err != nil && matchesError(err, api.ErrMustRetry); {
			if soc, err = v.Soc(); err != nil {
				if time.Since(start) > time.Minute {
					err = os.ErrDeadlineExceeded
				} else {
					time.Sleep(5 * time.Second)
				}
			}
		}

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(int(math.Round(soc)))

	default:
		log.Fatal("invalid action:", action)
	}
}
