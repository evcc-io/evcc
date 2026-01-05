package modbus

import (
	"github.com/andig/mbserver"
)

type DevicesSimulatorHandler struct {
	Devices map[uint8]struct {
		Coils    map[uint16]bool
		Discrete map[uint16]bool
		Input    map[uint16]uint16
		Holding  map[uint16]uint16
	}
}

func (th *DevicesSimulatorHandler) HandleCoils(req *mbserver.CoilsRequest) (res []bool, err error) {
	device, ok := th.Devices[req.UnitId]
	if !ok {
		// only reply to known unit IDs
		err = mbserver.ErrIllegalFunction
		return
	}

	for i := 0; i < int(req.Quantity); i++ {
		address := req.Addr + uint16(i)
		if _, ok := device.Coils[address]; !ok {
			err = mbserver.ErrIllegalDataAddress
			return
		}
		if req.IsWrite {
			device.Coils[address] = req.Args[i]
		}
		res = append(res, device.Coils[address])
	}
	return
}

func (th *DevicesSimulatorHandler) HandleDiscreteInputs(req *mbserver.DiscreteInputsRequest) (res []bool, err error) {
	device, ok := th.Devices[req.UnitId]
	if !ok {
		// only reply to known unit IDs
		err = mbserver.ErrIllegalFunction
		return
	}

	for i := 0; i < int(req.Quantity); i++ {
		address := req.Addr + uint16(i)
		if _, ok := device.Discrete[address]; !ok {
			err = mbserver.ErrIllegalDataAddress
			return
		}
		res = append(res, device.Discrete[address])
	}
	return
}

func (th *DevicesSimulatorHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) (res []uint16, err error) {
	device, ok := th.Devices[req.UnitId]
	if !ok {
		// only reply to known unit IDs
		err = mbserver.ErrIllegalFunction
		return
	}

	for i := 0; i < int(req.Quantity); i++ {
		address := req.Addr + uint16(i)
		if _, ok := device.Holding[address]; !ok {
			err = mbserver.ErrIllegalDataAddress
			return
		}
		if req.IsWrite {
			device.Holding[address] = req.Args[i]
		}
		res = append(res, device.Holding[address])
	}
	return
}

func (th *DevicesSimulatorHandler) HandleInputRegisters(req *mbserver.InputRegistersRequest) (res []uint16, err error) {
	device, ok := th.Devices[req.UnitId]
	if !ok {
		// only reply to known unit IDs
		err = mbserver.ErrIllegalFunction
		return
	}

	for i := 0; i < int(req.Quantity); i++ {
		address := req.Addr + uint16(i)
		if _, ok := device.Input[address]; !ok {
			err = mbserver.ErrIllegalDataAddress
			return
		}
		res = append(res, device.Input[address])
	}
	return
}
