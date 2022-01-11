package tasks

import (
	"errors"
	"net"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/util"
)

const Mqtt TaskType = "mqtt"

func init() {
	registry.Add(Mqtt, MqttHandlerFactory)
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

func (h *MqttHandler) Test(log *util.Logger, in ResultDetails) []ResultDetails {
	addr := net.JoinHostPort(in.IP, strconv.Itoa(h.Port))

	opt := mqtt.NewClientOptions()
	opt.AddBroker(addr)
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
				out := in.Clone()
				out.Topic = h.Topic
				return []ResultDetails{out}
			case <-timer.C:
				return nil
			}
		}
	}

	return nil
}
