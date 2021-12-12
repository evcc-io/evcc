package provider

import (
	"net"
	"strconv"
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
	// 	t.Error(err)
	// }

	// if err := server.Serve(); err != nil {
	// 	t.Error(err)
	// }

	// // TODO depends on https://github.com/mochi-co/mqtt/issues/6
	// broker := tcp.Listener().Addr().String()

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Error(err)
	}

	server := jeff.NewServer(l)
	// server.Dump = true
	server.Start()

	broker := l.Addr().String()

	pub, err := mqtt.NewClient(util.NewLogger("foo"), broker, "", "", "pub", 0, func(o *paho.ClientOptions) {
		o.SetProtocolVersion(3)
	})
	if err != nil {
		t.Error(err)
	}

	sub, err := mqtt.NewClient(util.NewLogger("foo"), broker, "", "", "sub", 0, func(o *paho.ClientOptions) {
		o.SetProtocolVersion(3)
	})
	if err != nil {
		t.Error(err)
	}
	_ = sub

	timerTopic := "timer"
	go func() {
		var i int
		for range time.Tick(time.Second) {
			i++
			t.Log("tick")
			if err := pub.Publish(timerTopic, false, strconv.Itoa(i)); err != nil {
				t.Error(err)
			}
		}
	}()

	// sub.Listen(topic, func(payload string) {
	// 	t.Log("recv:", payload)
	// })

	delay := 100 * time.Millisecond
	_ = delay

	// {
	// 	topic := "test1"
	// 	stringG := NewMqtt(util.NewLogger("foo"), sub, topic, 1, 0).StringGetter()

	// 	c := make(chan struct{})
	// 	go func() {
	// 		c <- struct{}{}
	// 		t.Log("startet")

	// 		if s, err := stringG(); err != nil {
	// 			t.Error(err)
	// 		} else if s != "hello" {
	// 			t.Error("wrong value", s)
	// 		}

	// 		close(c)
	// 		t.Log("stopped")
	// 	}()
	// 	<-c

	// 	// time.Sleep(2 * delay)

	// 	t.Log("publish")
	// 	if err := pub.Publish(topic, false, "hello"); err != nil {
	// 		t.Error(err)
	// 	}
	// 	<-c
	// }

	{
		topic := "test2"
		stringG := NewMqtt(util.NewLogger("foo"), sub, topic, 1, delay).StringGetter()

		c := make(chan struct{})
		go func() {
			c <- struct{}{}
			t.Log("startet")

			if _, err := stringG(); err == nil {
				t.Error("initial timeout error not received")
			}

			close(c)
			t.Log("stopped")
		}()
		<-c

		time.Sleep(2 * delay)

		t.Log("publish")
		if err := pub.Publish(topic, false, "hello"); err != nil {
			t.Error(err)
		}
		<-c
	}

	time.Sleep(1 * time.Second)
}
