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

func (th *DevicesSimulatorHandler) HandleCoils(req *mbserver.CoilsRequest) ([]bool, error) {
	device, ok := th.Devices[req.UnitId]
	if !ok {
		// only reply to known unit IDs
		return nil, mbserver.ErrIllegalFunction
	}

	var res []bool
	for i := 0; i < int(req.Quantity); i++ {
		address := req.Addr + uint16(i)
		if _, ok := device.Coils[address]; !ok {
			return nil, mbserver.ErrIllegalDataAddress
		}
		if req.IsWrite {
			device.Coils[address] = req.Args[i]
		}
		res = append(res, device.Coils[address])
	}
	return res, nil
}

func (th *DevicesSimulatorHandler) HandleDiscreteInputs(req *mbserver.DiscreteInputsRequest) ([]bool, error) {
	device, ok := th.Devices[req.UnitId]
	if !ok {
		// only reply to known unit IDs
		return nil, mbserver.ErrIllegalFunction
	}

	var res []bool
	for i := 0; i < int(req.Quantity); i++ {
		address := req.Addr + uint16(i)
		if _, ok := device.Discrete[address]; !ok {
			return nil, mbserver.ErrIllegalDataAddress
		}
		res = append(res, device.Discrete[address])
	}
	return res, nil
}

func (th *DevicesSimulatorHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	device, ok := th.Devices[req.UnitId]
	if !ok {
		// only reply to known unit IDs
		return nil, mbserver.ErrIllegalFunction
	}

	var res []uint16
	for i := 0; i < int(req.Quantity); i++ {
		address := req.Addr + uint16(i)
		if _, ok := device.Holding[address]; !ok {
			return nil, mbserver.ErrIllegalDataAddress
		}
		if req.IsWrite {
			device.Holding[address] = req.Args[i]
		}
		res = append(res, device.Holding[address])
	}
	return res, nil
}

func (th *DevicesSimulatorHandler) HandleInputRegisters(req *mbserver.InputRegistersRequest) ([]uint16, error) {
	device, ok := th.Devices[req.UnitId]
	if !ok {
		// only reply to known unit IDs
		return nil, mbserver.ErrIllegalFunction
	}

	var res []uint16
	for i := 0; i < int(req.Quantity); i++ {
		address := req.Addr + uint16(i)
		if _, ok := device.Input[address]; !ok {
			return nil, mbserver.ErrIllegalDataAddress
		}
		res = append(res, device.Input[address])
	}
	return res, nil
}
