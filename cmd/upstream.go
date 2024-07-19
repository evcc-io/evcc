package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/andig/wsp/client"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

func upstream(reverseProxyUrl, socketProxyUrl string, ch chan util.Param) {
	client.NewClient(&client.Config{
		ID:           uuid.NewString(),
		Targets:      []string{reverseProxyUrl},
		PoolIdleSize: 2,
		PoolMaxSize:  10,
	}).Start(context.Background())

	for {
		if err := connectService(socketProxyUrl, ch); err != nil {
			fmt.Println("ws connect:", err)
			time.Sleep(time.Second)
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
