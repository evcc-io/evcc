package provider

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
	jeff "github.com/jeffallen/mqtt"
)

func init() {
	util.LogLevel("fatal", nil)
}

const (
	mqttTimeout = 100 * time.Millisecond
)

func TestMain(t *testing.M) {
	util.WaitInitialTimeout = 3 * mqttTimeout
	os.Exit(t.Run())
}

func server(t *testing.T) (net.Listener, *mqtt.Client, *mqtt.Client) {
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

	return l, pub, sub
}

func TestMqttReceiveInInitialTimeout(t *testing.T) {
	l, pub, sub := server(t)
	defer l.Close()

	topic := "test"
	stringG := NewMqtt(util.NewLogger("foo"), sub, topic, 1, 0).StringGetter()

	c := make(chan struct{})

	go func() {
		c <- struct{}{}

		if s, err := stringG(); err != nil {
			t.Error(err)
		} else if s != "hello" {
			t.Error("wrong value", s)
		}

		close(c)
	}()
	<-c

	t.Log("publish")
	if err := pub.Publish(topic, false, "hello"); err != nil {
		t.Error(err)
	}
	<-c
}

func TestMqttReceiveAfterInitialTimeout(t *testing.T) {
	l, pub, sub := server(t)
	defer l.Close()

	topic := "test"
	stringG := NewMqtt(util.NewLogger("foo"), sub, topic, 1, mqttTimeout).StringGetter()

	c := make(chan struct{})

	go func() {
		c <- struct{}{}

		if _, err := stringG(); err == nil {
			t.Error("initial timeout error not received")
		}

		close(c)
	}()
	<-c

	time.Sleep(util.WaitInitialTimeout + 2*mqttTimeout)

	if err := pub.Publish(topic, false, "hello"); err != nil {
		t.Error(err)
	}
	<-c
}

func startTicker(t *testing.T, ticker string, pub *mqtt.Client) chan struct{} {
	// timeout ticker
	tC := make(chan struct{})
	go func() {
		close(tC)
		var i int
		for ; true; <-time.Tick(mqttTimeout) {
			i++

			// 2 * util.WaitInitialTimeout+ 50%
			if i > 15 {
				t.Error("timeout")
				break
			}

			t.Log("tick")
			if err := pub.Publish(ticker, false, strconv.Itoa(i)); err != nil {
				t.Error(err)
			}
		}
	}()

	return tC
}

func TestMqttReceiveTickerInInitialTimeout(t *testing.T) {
	l, pub, sub := server(t)
	defer l.Close()

	ticker := "timer"
	<-startTicker(t, ticker, pub)

	// timeout handler
	log := util.NewLogger("foo")
	timer := NewMqtt(log, sub, ticker, 1, 2*mqttTimeout).StringGetter()

	stringGFactory := func(topic string) func() (string, error) {
		g := NewMqtt(log, sub, topic, 1, 0).StringGetter()
		return func() (val string, err error) {
			if val, err = g(); err == nil {
				_, err = timer()
			}
			return val, err
		}
	}

	topic := "test"
	stringG := stringGFactory(topic)

	c := make(chan struct{})

	// receive initial value within initialTimeout
	go func() {
		c <- struct{}{}

		if s, err := stringG(); err != nil {
			t.Error(err)
		} else if s != "hello 1" {
			t.Error("wrong value", s)
		}

		t.Log("received 1")

		close(c)
	}()
	<-c

	time.Sleep(2 * mqttTimeout)

	if err := pub.Publish(topic, true, "hello 1"); err != nil {
		t.Error(err)
	}
	<-c

	// receive second value after initialTimeout
	c = make(chan struct{})

	go func() {
		c <- struct{}{}

		for {
			s, err := stringG()

			// keep waiting for updated value
			if err == nil && s == "hello 1" {
				t.Log("continue")
				time.Sleep(mqttTimeout / 3)
				continue
			}

			// verify updated value
			if err == nil && s != "hello 2" {
				err = fmt.Errorf("wrong value: %v", s)
			}

			if err != nil {
				t.Error(err)
			}

			break
		}

		close(c)
	}()
	<-c

	t.Log("waiting 2")
	time.Sleep(util.WaitInitialTimeout + 2*mqttTimeout)

	if err := pub.Publish(topic, false, "hello 2"); err != nil {
		t.Error(err)
	}
	<-c
}
func TestMqttReceiveTickerAfterInitialTimeout(t *testing.T) {
	l, pub, sub := server(t)
	defer l.Close()

	ticker := "timer"
	<-startTicker(t, ticker, pub)

	// timeout handler
	log := util.NewLogger("foo")
	timer := NewMqtt(log, sub, ticker, 1, 2*mqttTimeout).StringGetter()

	stringGFactory := func(topic string) func() (string, error) {
		g := NewMqtt(log, sub, topic, 1, 0).StringGetter()
		return func() (val string, err error) {
			if val, err = g(); err == nil {
				_, err = timer()
			}
			return val, err
		}
	}

	topic := "test"
	stringG := stringGFactory(topic)

	c := make(chan struct{})

	// receive initial value within initialTimeout
	go func() {
		c <- struct{}{}

		if _, err := stringG(); err == nil {
			t.Error("initial timeout error not received")
		}

		t.Log("received")

		close(c)
	}()
	<-c

	time.Sleep(util.WaitInitialTimeout + 2*mqttTimeout)

	if err := pub.Publish(topic, true, "hello"); err != nil {
		t.Error(err)
	}
	<-c
}
