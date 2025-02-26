package main

import (
	"fmt"
	"time"

	"github.com/mlnoga/rct"
)

func run() (any, error) {
	conn, err := rct.NewConnection("localhost", time.Second)
	if err != nil {
		return nil, err
	}

	for {
		// var dg *rct.Datagram
		dg, err := conn.Receive()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("%+v\n", dg)
	}
}

func main() {
	if _, err := run(); err != nil {
		panic(err)
	}
}
