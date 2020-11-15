package cmd

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/andig/evcc/util"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

func init() {
	registry.Add("modbus", ModbusHandlerFactory)
}

func ModbusHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := ModbusHandler{
		Port:    502,
		IDs:     []uint8{1},
		Models:  []int{1},
		Point:   "Md", // Model
		Timeout: timeout,
	}

	err := util.DecodeOther(conf, &handler)

	if err == nil && len(handler.IDs) == 0 {
		err = errors.New("missing slave IDs")
	}

	return &handler, err
}

type ModbusHandler struct {
	Port    int
	IDs     []uint8
	Models  []int
	Point   string
	Timeout time.Duration
}

func (h *ModbusHandler) Test(ip net.IP) bool {
	addr := fmt.Sprintf("%s:%d", ip.String(), h.Port)
	conn := meters.NewTCP(addr)
	dev := sunspec.NewDevice("sunspec")

	conn.Timeout(timeout)

	for idx, slaveID := range h.IDs {
		// grace period for id switch
		if idx > 0 {
			time.Sleep(100 * time.Millisecond)
		}

		conn.Slave(slaveID)
		err := dev.Initialize(conn.ModbusClient())

		if errors.Is(err, meters.ErrPartiallyOpened) {
			err = nil
		}
		if err != nil {
			continue
		}

		if len(h.Models) == 0 {
			return true
		}

		for _, model := range h.Models {
			_, res, err := dev.QueryPointAny(
				conn.ModbusClient(),
				model,
				0,
				h.Point,
			)

			if err == nil {
				fmt.Printf("modbus: %d.%s/%v %+v\n", model, h.Point, res.Value(), res)
				return true
			}
		}

		return false
	}

	return false
}
