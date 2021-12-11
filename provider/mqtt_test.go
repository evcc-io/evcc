package provider

import (
	"testing"

	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
	mochi "github.com/mochi-co/mqtt/server"
	"github.com/mochi-co/mqtt/server/listeners"
)

func TestMqttInitialTimeout(t *testing.T) {
	server := mochi.New()
	tcp := listeners.NewTCP("t1", "")

	// Add the listener to the server with default options (nil).
	if err := server.AddListener(tcp, nil); err != nil {
		t.Fatal(err)
	}

	if err := server.Serve(); err != nil {
		t.Fatal(err)
	}

	broker := tcp.Listener().Addr().String()
	client, err := mqtt.NewClient(util.NewLogger("foo"), broker, "", "", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	_ = client

}
