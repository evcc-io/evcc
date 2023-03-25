package main

import (
	"fmt"
	"net"

	"github.com/andig/mbserver"
	gridx "github.com/grid-x/modbus"
)

func main() {
	a, err := net.ResolveUDPAddr("udp4", ":502")
	if err != nil {
		fmt.Println(err)
		return
	}

	l, err := net.ListenUDP("udp", a)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	h := &mbserver.DummyHandler{}

	srv, err := mbserver.New(h)
	if err != nil {
		panic(err)
	}

	if err := srv.Start(l); err != nil {
		panic(err)
	}

	u := gridx.UDPClient("localhost:502")
	b, err := u.ReadHoldingRegisters(0, 1)
	if err != nil {
		panic(err)
	}

	fmt.Println(b)
}
