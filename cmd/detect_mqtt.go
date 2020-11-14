package cmd

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/andig/evcc/util"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func init() {
	registry.Add("mqtt", MqttHandlerFactory)
}

func MqttHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := MqttHandler{
		Port:    1883,
		Timeout: timeout,
	}
	err := util.DecodeOther(conf, &handler)

	if err == nil && handler.Port == 0 {
		err = errors.New("missing port")
	}

	return &handler, err
}

type MqttHandler struct {
	Port    int
	Topic   string
	Timeout time.Duration
}

func (h *MqttHandler) Test(ip net.IP) bool {
	broker := fmt.Sprintf("%s:%d", ip.String(), h.Port)

	opt := mqtt.NewClientOptions()
	opt.AddBroker(broker)
	opt.SetConnectTimeout(timeout)

	client := mqtt.NewClient(opt)

	var ok bool
	token := client.Connect()
	if token.Wait() {
		ok = token.Error() == nil
	}

	if ok && h.Topic != "" {
		recv := make(chan bool, 1)
		_ = client.Subscribe(h.Topic, 1, func(mqtt.Client, mqtt.Message) {
			recv <- true
		})

		timer := time.NewTimer(timeout)
		for {
			select {
			case <-recv:
				break
			case <-timer.C:
				ok = false
			}
		}
	}

	return ok
}
