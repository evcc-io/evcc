package provider

import (
	"net"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
	jeff "github.com/jeffallen/mqtt"
)

func TestMqttInitialTimeout(t *testing.T) {
	// server := mochi.New()
	// tcp := listeners.NewTCP("t1", "")

	// // Add the listener to the server with default options (nil).
	// if err := server.AddListener(tcp, nil); err != nil {
	// 	t.Fatal(err)
	// }

	// if err := server.Serve(); err != nil {
	// 	t.Fatal(err)
	// }

	// // TODO depends on https://github.com/mochi-co/mqtt/issues/6
	// broker := tcp.Listener().Addr().String()

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	server := jeff.NewServer(l)
	server.Start()

	broker := l.Addr().String()

	client, err := mqtt.NewClient(util.NewLogger("foo"), broker, "foo", "bar", "cid", 0, func(o *paho.ClientOptions) {
		o.SetProtocolVersion(3)
	})
	if err != nil {
		t.Fatal(err)
	}

	topic := "test"
	client.Listen(topic, func(payload string) {
		t.Log("recv:", payload)
	})

	client.Publish(topic, false, "hello")
	time.Sleep(1 * time.Second)
}
