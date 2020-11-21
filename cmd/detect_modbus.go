package cmd

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
	"github.com/andig/gosunspec/models/model1"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

func init() {
	registry.Add("modbus", ModbusHandlerFactory)
}

func ModbusHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := ModbusHandler{
		Port:    502,
		Min:     1,
		Max:     5,
		Timeout: 100 * time.Millisecond,
	}

	err := util.DecodeOther(conf, &handler)

	if err == nil && handler.Max == 0 {
		err = errors.New("missing max")
	}

	return &handler, err
}

type ModbusHandler struct {
	Port     int
	Min, Max uint8
	Timeout  time.Duration
}

func (h *ModbusHandler) Test(ip net.IP) bool {
	for slaveID := h.Min; slaveID <= h.Max; slaveID++ {
		addr := fmt.Sprintf("%s:%d", ip.String(), h.Port)
		conn, err := modbus.NewConnection(addr, "", "", 0, false, slaveID)
		if err != nil {
			continue
		}

		dev, err := modbus.NewDevice("sunspec", 0, false)
		if err != nil {
			continue
		}

		err = dev.Initialize(conn)
		if errors.Is(err, meters.ErrPartiallyOpened) {
			err = nil
		}
		if err != nil {
			continue
		}

		res, err := dev.(*sunspec.SunSpec).QueryPoint(
			conn,
			model1.ModelID,
			0,
			model1.Md,
		)
		fmt.Println(res)

		if err != nil {
			continue
		}

		return true
	}

	return false
}
