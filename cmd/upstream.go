package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"nhooyr.io/websocket"
)

func upstream(service string, ch <-chan util.Param) {
	for {
		if err := connectService(service, ch); err != nil {
			time.Sleep(time.Second)
			fmt.Println("ws connect:", err)
		}
	}
}

func connectService(service string, ch <-chan util.Param) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, service, nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow()

	for p := range ch {
		msg := "{" + server.Kv(p) + "}"
		if err := conn.Write(context.Background(), websocket.MessageText, []byte(msg)); err != nil {
			return err
		}
	}

	return nil
}
