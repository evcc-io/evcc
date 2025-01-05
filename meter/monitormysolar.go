package meter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("monitormysolar", NewMonitorMySolarFromConfig)
}

type UpdateCommand struct {
	Setting string `json:"setting"`
	From    string `json:"from"`
}

type UpdateCommandString struct {
	*UpdateCommand
	Value string `json:"value"`
}

type UpdateCommandFloat struct {
	*UpdateCommand
	Value float64 `json:"value"`
}

type CommandResponse struct {
	Status string `json:"status"`
}

type MonitorMySolar struct {
	log        *util.Logger
	usage      string
	dongleId   string
	client     *mqtt.Client
	respChan   chan CommandResponse
	inputBank1 *util.Monitor[InputBank1]
	holdBank2  *util.Monitor[HoldBank2]
}

type InputBank1 struct {
	Payload struct {
		Ptouser    float64 `json:"Ptouser"`
		Ptogrid    float64 `json:"Ptogrid"`
		Pall       float64 `json:"Pall"`
		Pcharge    float64 `json:"Pcharge"`
		Pdischarge float64 `json:"Pdischarge"`
		SOC        float64 `json:"SOC"`
	} `json:"payload"`
}

type HoldBank2 struct {
	Payload struct {
		ACChgPowerCMD float64 `json:"ACChgPowerCMD"`
	} `json:"payload"`
}

func NewMonitorMySolarFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		DongleId    string
		Timeout     time.Duration
		Usage       string
		capacity    `mapstructure:",squash"`
	}{
		Timeout: 15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.DongleId == "" {
		return nil, fmt.Errorf("dongleId is required")
	}

	switch cc.Usage {
	case "grid", "pv", "battery":
	default:
		return nil, fmt.Errorf("invalid usage: %s", cc.Usage)
	}

	log := util.NewLogger("monitormysolar")

	client, err := mqtt.RegisteredClientOrDefault(log, cc.Config)
	if err != nil {
		return nil, err
	}

	mms := &MonitorMySolar{
		log:        log,
		dongleId:   cc.DongleId,
		usage:      cc.Usage,
		client:     client,
		respChan:   make(chan CommandResponse, 1),
		inputBank1: util.NewMonitor[InputBank1](cc.Timeout),
		holdBank2:  util.NewMonitor[HoldBank2](cc.Timeout),
	}

	if err := client.Listen(fmt.Sprintf("%s/inputbank1", cc.DongleId), func(data string) {
		handleMessage(mms.inputBank1, data, log)
	}); err != nil {
		return nil, err
	}

	if err := client.Listen(fmt.Sprintf("%s/holdbank2", cc.DongleId), func(data string) {
		handleMessage(mms.holdBank2, data, log)
	}); err != nil {
		return nil, err
	}

	var currents func() (float64, float64, float64, error)
	var soc func() (float64, error)
	var capacity func() float64
	var setBatteryMode func(api.BatteryMode) error

	if mms.usage == "battery" {
		soc = mms.soc
		setBatteryMode = mms.setBatteryMode
		if err := client.Listen(fmt.Sprintf("%s/response", cc.DongleId), mms.responseHandler); err != nil {
			return nil, err
		}
	}

	m, err := NewConfigurable(mms.power)
	if err != nil {
		return nil, err
	}

	res := m.Decorate(nil, currents, nil, nil, soc, capacity, nil, setBatteryMode)
	return res, nil
}

func (mms *MonitorMySolar) power() (float64, error) {
	value, err := mms.inputBank1.Get()
	if err != nil {
		return 0, err
	}
	switch mms.usage {
	case "grid":
		return value.Payload.Ptouser - value.Payload.Ptogrid, nil
	case "pv":
		return value.Payload.Pall, nil
	case "battery":
		return value.Payload.Pdischarge - value.Payload.Pcharge, nil
	}
	return 0, nil
}

func (mms *MonitorMySolar) soc() (float64, error) {
	value, err := mms.inputBank1.Get()
	if err != nil {
		return 0, err
	}
	return value.Payload.SOC, nil
}

func (mms *MonitorMySolar) setBatteryMode(mode api.BatteryMode) error {
	switch mode {
	case api.BatteryNormal:
		updates := []interface{}{
			NewUpdateCommand("DischgPowerPercentCMD", 100.0),
			NewUpdateCommand("ACChgStart2", "00:00:00"),
			NewUpdateCommand("ACChgEnd2", "00:00:00"),
			NewUpdateCommand("ACChgPowerCMD", 0.0),
		}
		if err := mms.sendUpdates(updates); err != nil {
			return err
		}
	case api.BatteryHold:
		updates := []interface{}{
			NewUpdateCommand("DischgPowerPercentCMD", 0.0),
			NewUpdateCommand("ACChgStart2", "00:00:00"),
			NewUpdateCommand("ACChgEnd2", "00:00:00"),
			NewUpdateCommand("ACChgPowerCMD", 0.0),
		}
		if err := mms.sendUpdates(updates); err != nil {
			return err
		}
	case api.BatteryCharge:
		updates := []interface{}{
			NewUpdateCommand("DischgPowerPercentCMD", 0.0),
			NewUpdateCommand("ACChgStart2", "00:00:00"),
			NewUpdateCommand("ACChgEnd2", "23:59:59"),
			NewUpdateCommand("ACChgPowerCMD", 100.0),
		}
		if err := mms.sendUpdates(updates); err != nil {
			return err
		}
	case api.BatteryUnknown:
		return fmt.Errorf("invalid battery mode: %s", mode)
	}
	return nil
}

func (mms *MonitorMySolar) sendUpdates(updates []interface{}) error {
	for _, update := range updates {
		if err := mms.sendUpdate(update); err != nil {
			return err
		}
	}
	return nil
}

func (mms *MonitorMySolar) sendUpdate(update interface{}) error {
	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("error marshaling command: %w", err)
	}

	for len(mms.respChan) > 0 {
		<-mms.respChan
	}

	mms.log.DEBUG.Printf("sending update: %s", string(data))
	if err := mms.client.Publish(fmt.Sprintf("%s/update", mms.dongleId), false, string(data)); err != nil {
		return fmt.Errorf("error publishing command: %w", err)
	}

	select {
	case resp := <-mms.respChan:
		if resp.Status != "success" {
			return fmt.Errorf("command failed with status: %s", resp.Status)
		}
		mms.log.DEBUG.Printf("command response: %+v", resp)
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for command response")
	}

	return nil
}

func handleMessage[T InputBank1 | HoldBank2](monitor *util.Monitor[T], data string, log *util.Logger) {
	var res T
	if err := json.Unmarshal([]byte(data), &res); err != nil {
		log.ERROR.Printf("error parsing response: %v", err)
		return
	}
	monitor.Set(res)
}

func (mms *MonitorMySolar) responseHandler(data string) {
	var resp CommandResponse
	mms.log.DEBUG.Printf("received response: %s", data)
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		mms.log.ERROR.Printf("error parsing response: %v", err)
		resp = CommandResponse{Status: "error"}
	}

	mms.respChan <- resp
}

func NewUpdateCommand[T string | float64](setting string, value T) interface{} {
	base := &UpdateCommand{
		Setting: setting,
		From:    "evcc",
	}

	switch v := any(value).(type) {
	case string:
		return &UpdateCommandString{
			UpdateCommand: base,
			Value:         v,
		}
	case float64:
		return &UpdateCommandFloat{
			UpdateCommand: base,
			Value:         v,
		}
	default:
		return nil
	}
}
