package solarman

import (
	"encoding/binary"
)

type SolarmanRequest struct {
	SlaveID byte
	header  []byte
	payload []byte
}

type RequestBuilder interface {
	SetLoggerSerial(LoggerSerial uint32) *SolarmanRequestBuilder
	ReadHoldingRegistersRequest(address uint16, quantity uint16) []byte
	ReadInputRegistersRequest(address uint16, quantity uint16) []byte
	ReadCoilsRequest(address uint16, quantity uint16) []byte
	WriteSingleCoilRequest(address uint16, value uint16) []byte
	WriteSingleRegisterRequest(address uint16, value uint16) []byte
	WriteMultipleRegistersRquest(address uint16, length uint16, values []byte) []byte
}

type SolarmanRequestBuilder struct {
	Request *SolarmanRequest
}

func NewSolarmanRequestBuilder(SlaveID byte, Serial byte) RequestBuilder {
	header := []byte{
		0xA5,       //Start 0
		0x00, 0x00, //Length (placeholder) 1,2
		0x10, 0x45, //Control Code 3,4
		Serial, 0x00, //Serial, 5,6
		0x00, 0x00, 0x00, 0x00, //LoggerSerial (placeholder) 7,8,9,10
	}

	payload := []byte{
		0x02,       //Frame Type 11
		0x00, 0x00, //Sensor Type 12,13
		0x00, 0x00, 0x00, 0x00, //Total Working Time 14,15,16,17
		0x00, 0x00, 0x00, 0x00, //Power on Time 18,19,20,21
		0x00, 0x00, 0x00, 0x00, //Offset Time 22,23,24,25
	}

	return &SolarmanRequestBuilder{
		Request: &SolarmanRequest{
			header:  header,
			payload: payload,
			SlaveID: SlaveID,
		},
	}
}

func (srb *SolarmanRequestBuilder) createSingleValueRequest(funcCode byte, address uint16, value uint16) []byte {
	length := len(srb.Request.header) + len(srb.Request.payload)
	request := make([]byte, length+10)
	copy(request, srb.Request.header)
	copy(request[len(srb.Request.header):], srb.Request.payload)

	pdu := make([]byte, 8)
	pdu[0] = srb.Request.SlaveID
	pdu[1] = funcCode
	pdu[2], pdu[3] = splitUInt16(address, binary.BigEndian)
	pdu[4], pdu[5] = splitUInt16(value, binary.BigEndian)

	pdu[6], pdu[7] = CRC(pdu[0:6])
	copy(request[length:], pdu)
	request[1], request[2] = splitUInt16(uint16(len(srb.Request.payload)+8), binary.LittleEndian)
	request[length+8] = checkSum(request[1 : len(request)-2])
	request[length+9] = 0x15
	return request
}

func (srb *SolarmanRequestBuilder) createMultipleValuesRequest(funcCode byte, address uint16, quantity uint16, values []byte) []byte {
	length := len(srb.Request.header) + len(srb.Request.payload)
	pdu_length := 7 + int(len(values)*2) + 2

	pdu := make([]byte, pdu_length)
	pdu[0] = srb.Request.SlaveID
	pdu[1] = funcCode
	pdu[2], pdu[3] = splitUInt16(address, binary.BigEndian)
	pdu[4], pdu[5] = splitUInt16(quantity, binary.BigEndian)
	pdu[6] = uint8(len(values) * 2)
	for i, v := range values {
		pdu[8+(i*2)] = v
	}
	pdu[7+len(values)*2], pdu[8+len(values)*2] = CRC(pdu[0 : 7+len(values)*2])

	request := make([]byte, length+pdu_length+2)
	copy(request, srb.Request.header)
	copy(request[len(srb.Request.header):], srb.Request.payload)
	copy(request[length:], pdu)
	request[1], request[2] = splitUInt16(uint16(len(srb.Request.payload)+pdu_length), binary.LittleEndian)
	request[length+pdu_length] = checkSum(request[1 : len(request)-2])
	request[length+pdu_length+1] = 0x15

	return request
}

func (srb *SolarmanRequestBuilder) ReadHoldingRegistersRequest(address uint16, quantity uint16) []byte {
	return srb.createSingleValueRequest(0x03, address, quantity)
}

func (srb *SolarmanRequestBuilder) ReadInputRegistersRequest(address uint16, quantity uint16) []byte {
	return srb.createSingleValueRequest(0x04, address, quantity)
}

func (srb *SolarmanRequestBuilder) ReadCoilsRequest(address uint16, quantity uint16) []byte {
	return srb.createSingleValueRequest(0x01, address, quantity)
}

func (srb *SolarmanRequestBuilder) WriteSingleCoilRequest(address uint16, value uint16) []byte {
	return srb.createSingleValueRequest(0x05, address, value)
}

func (srb *SolarmanRequestBuilder) WriteSingleRegisterRequest(address uint16, value uint16) []byte {
	return srb.createSingleValueRequest(0x06, address, value)
}

func (srb *SolarmanRequestBuilder) WriteMultipleRegistersRquest(address uint16, quantity uint16, values []byte) []byte {
	return srb.createMultipleValuesRequest(0x10, address, quantity, values)
}

func (srb *SolarmanRequestBuilder) SetLoggerSerial(LoggerSerial uint32) *SolarmanRequestBuilder {
	srb.Request.header[7], srb.Request.header[8], srb.Request.header[9], srb.Request.header[10] = splitUInt32(LoggerSerial, binary.LittleEndian)
	return srb
}

func splitUInt16(i uint16, byteorder binary.ByteOrder) (byte, byte) {
	bytes := make([]byte, 2)
	byteorder.PutUint16(bytes, i)
	return bytes[0], bytes[1]
}

func splitUInt32(i uint32, byteorder binary.ByteOrder) (byte, byte, byte, byte) {
	bytes := make([]byte, 4)
	byteorder.PutUint32(bytes, i)
	return bytes[0], bytes[1], bytes[2], bytes[3]
}

func checkSum(b []byte) byte {
	checksum := byte(0)
	for _, v := range b {
		checksum += v & 0xFF
	}
	return checksum & 0xFF
}
