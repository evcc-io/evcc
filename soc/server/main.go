package main

import (
	"log"

	compile "github.com/andig/evcc/server"
	"github.com/andig/evcc/soc/server/server"
	"github.com/andig/evcc/soc/server/ui"
)

func main() {
	log.Printf("soc-server %s (%s)", compile.Version, compile.Commit)

	go ui.Run()
	go server.Run()

	select {}
}
