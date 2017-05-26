package modbus

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	// ReadCoils is Modbus function code 1.
	ReadCoils uint8 = iota + 1

	// ReadDiscreteInputs is Modbus function code 2.
	ReadDiscreteInputs

	// ReadHoldingRegisters is Modbus function code 3.
	ReadHoldingRegisters

	// ReadInputRegisters is Modbus function code 4.
	ReadInputRegisters

	// WriteSingleCoil is Modbus function code 5.
	WriteSingleCoil

	// WriteSingleRegister is Modbus function code 6.
	WriteSingleRegister

	// WriteMultipleCoils is Modbus function code 15.
	WriteMultipleCoils = 15

	// WriteMultipleRegisters is Modbus function code 16.
	WriteMultipleRegisters
)

const (
	// IllegalFunction is Modbus exception code 1.
	IllegalFunction uint8 = iota + 1

	// IllegalAddress is Modbus exception code 2.
	IllegalAddress

	// IllegalDataValue is Modbus exception code 3.
	IllegalDataValue

	// SlaveDeviceFailure is Modbus exception code 4.
	SlaveDeviceFailure

	// Acknowledge is Modbus exception code 5.
	Acknowledge

	// SlaveDeviceBusy is Modbus exception code 6.
	SlaveDeviceBusy

	// NegativeAcknowledge is Modbus exception code 7.
	NegativeAcknowledge

	// MemoryParityError is Modbus exception code 8.
	MemoryParityError

	// GatewayPathUnavailable is Modbus exception code 10.
	GatewayPathUnavailable = 10

	// GatewayTargetDeviceFailedToRespond is Modbus exception code 11.
	GatewayTargetDeviceFailedToRespond
)

// MBAP is the Modbus Application Header. Only Modbus TCP/IP message have an
// MBAP header. The MBAP header has 4 fields with a total length of 7 bytes.
type MBAP struct {
	// TransactionID identifies a request/response transaction.
	TransactionID uint16

	// ProtocolID defines the protocol, for Modbus it's always 0.
	ProtocolID uint16

	// Length shows how much bytes are following.
	Length uint16

	// UnitID or slave id identifies a slave.
	UnitID uint8
}

// UnmarshalBinary unmarshals a binary represention of MBAP.
func (m *MBAP) UnmarshalBinary(b []byte) error {
	if len(b) != 7 {
		return fmt.Errorf("failed to unmarshal byte slice to MBAP: byte slice has invalid length of %d", len(b))
	}

	m.TransactionID = binary.BigEndian.Uint16(b[:2])
	m.ProtocolID = binary.BigEndian.Uint16(b[2:4])
	m.Length = binary.BigEndian.Uint16(b[4:6])
	m.UnitID = uint8(b[6])

	return nil
}

// MarshalBinary marshals a MBAP to it binary form.
func (m *MBAP) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	data := []interface{}{
		m.TransactionID,
		m.ProtocolID,
		m.Length,
		m.UnitID,
	}

	for _, v := range data {
		err := binary.Write(buf, binary.BigEndian, v)

		if err != nil {
			return buf.Bytes(), fmt.Errorf("failed to marshal MBAP to binary form: %v", err)
		}

	}

	return buf.Bytes(), nil
}

// Request is a Modbus request.
type Request struct {
	MBAP

	FunctionCode uint8
	Data         []byte
}

// UnmarshalBinary unmarshals binary represention of Request.
func (r *Request) UnmarshalBinary(b []byte) error {
	if err := r.MBAP.UnmarshalBinary(b[0:7]); err != nil {
		return err
	}

	r.FunctionCode = uint8(b[7])
	r.Data = b[8:]

	return nil
}

// Response is a Modbus response.
type Response struct {
	MBAP
	FunctionCode uint8
	Data         []byte

	exception bool
}

// NewResponse creates a Response for a Request.
func NewResponse(r Request, data []byte) *Response {
	resp := &Response{
		MBAP:         r.MBAP,
		FunctionCode: r.FunctionCode,
		Data:         data,
	}

	resp.MBAP.Length = uint16(len(data) + 3)
	return resp
}

// NewExceptionResponse creates a expection response.
func NewExceptionResponse(r Request, code uint8) *Response {
	resp := &Response{
		MBAP:         r.MBAP,
		FunctionCode: r.FunctionCode + 0x80,
		Data:         []byte{byte(code)},
		exception:    true,
	}

	resp.MBAP.Length = 3
	return resp
}

// MarshalBinary marshals a Response to it binary form.
func (r *Response) MarshalBinary() ([]byte, error) {
	mbap, err := r.MBAP.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response to its binary form: %v", err)
	}

	pdu := new(bytes.Buffer)
	data := []interface{}{
		r.FunctionCode,
	}
	if !r.exception {
		data = append(data, uint8(len(r.Data)))
	}
	data = append(data, r.Data)

	for _, v := range data {
		err := binary.Write(pdu, binary.BigEndian, v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response to its binary form: %v", err)
		}
	}

	return append(mbap, pdu.Bytes()...), nil
}
