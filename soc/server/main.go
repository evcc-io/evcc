package main

import (
	"github.com/andig/evcc/soc/server/server"
	"github.com/andig/evcc/soc/server/ui"
)

func main() {
	// fmt.Println(auth.AuthorizedToken("Johnny Cash", "demo"))

	go ui.Run()
	go server.Run()

	select {}
}
