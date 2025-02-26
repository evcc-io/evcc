package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mlnoga/rct"
)

type item struct {
	updated time.Time
	dg      *rct.Datagram
}

func runChan() (any, error) {
	conn, err := rct.NewConnection("localhost", time.Second)
	if err != nil {
		return nil, err
	}

	var mu sync.RWMutex
	cache := make(map[rct.Identifier]item)

	cconn := conn.Conn()
	buf := make([]byte, 1024)

	ctx := context.Background()

	done := make(chan error, 1)
	bufC := make(chan byte, 1024)
	dgC := make(chan rct.Datagram)

	go func() {
		var rdb rct.DatagramBuilder

		for ctx.Err() == nil {
			rdb.Build(&rct.Datagram{rct.Read, rct.TotalGridPowerW, nil})

			if _, err := cconn.Write(rdb.Bytes()); err != nil {
				panic("write:" + err.Error())
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	go rct.ParseAsync(ctx, bufC, dgC)

	go func() {
		for {
			if err := ctx.Err(); err != nil {
				done <- err
				return
			}

			n, err := cconn.Read(buf)
			if err != nil {
				done <- err
				return
			}

			fmt.Println("recv:", n, "-", fmt.Sprintf("% 0x", buf[:min(8, n)]))
			for _, b := range buf[:n] {
				bufC <- b
			}
		}
	}()

	for dg := range dgC {
		select {
		case err := <-done:
			return nil, err
		default:
		}

		fmt.Printf("data: %+v\n", dg)

		if dg.Cmd == rct.Response {
			mu.Lock()
			cache[dg.Id] = item{updated: time.Now(), dg: &dg}
			// fmt.Println("item:", len(cache))
			mu.Unlock()
		}
	}

	return nil, nil
}

func main() {
	if _, err := runChan(); err != nil {
		panic(err)
	}
}
