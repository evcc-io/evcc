package main

import (
	"flag"
	"log"

	"github.com/evcc-io/evcc/tariff/ekz"
)

func main() {
	port := flag.Int("port", 33927, "Port to run the mock EKZ API server on")
	flag.Parse()

	server := ekz.NewMockServer(*port)
	log.Fatal(server.Start())
}
