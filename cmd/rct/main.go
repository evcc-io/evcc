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

	cconn := conn.Conn()
	buf := make([]byte, 2048)

	for {
		n, err := cconn.Read(buf)
		if err != nil {
			return nil, err
		}
		fmt.Println("recv:", n, "-", fmt.Sprintf("% 0x", buf[:n]))

		if n == len(buf) {
			panic("large message")
		}

		var i int

		for i < n {
			// fmt.Println("i:", i, "n:", n, "len:", len(buf))
			dg, n, err := rct.Parse(buf[i:n])
			if err != nil {
				fmt.Println("data:", n, err)
				break
			}

			i += n

			fmt.Printf("data: %d - %+v\n", n, dg)
		}

		// os.Exit(0)
	}
}

func main() {
	if _, err := run(); err != nil {
		panic(err)
	}
}
