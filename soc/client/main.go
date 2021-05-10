package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"

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

	var key string
	for _, arg := range os.Args[2:] {
		if key == "" {
			key = strings.ToLower(strings.TrimLeft(arg, "-"))
		} else {
			if key == "log" {
				util.LogLevel(arg, nil)
			} else {
				params[key] = arg
			}
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

	soc, err := v.SoC()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(int(math.Round(soc)))
}
