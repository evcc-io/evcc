package ecoflow

import (
	"net"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A broker that answers only after the connect timed out must not get a
// session: the client gives up instead of connecting in the background.
func TestConnectTimeoutDisconnects(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	closed := make(chan struct{})
	connack := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		_, _ = conn.Read(buf) // CONNECT
		<-connack
		_, _ = conn.Write([]byte{0x20, 0x02, 0x00, 0x00}) // CONNACK accepted

		for {
			if _, err := conn.Read(buf); err != nil {
				close(closed)
				return
			}
		}
	}()

	client := paho.NewClient(paho.NewClientOptions().
		AddBroker("tcp://" + ln.Addr().String()).
		SetConnectTimeout(5 * time.Second).
		SetAutoReconnect(false))

	assert.Error(t, connect(client, 50*time.Millisecond))
	close(connack)

	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("client kept the late connection")
	}
	assert.False(t, client.IsConnected())
}
