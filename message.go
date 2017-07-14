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

// Error represesents a Modbus protocol error.
type Error struct {
	// Code contains the Modbus exception code.
	Code uint8
	msg  string
}

func (e Error) Error() string {
	return fmt.Sprintf("Modbus exception code: %d: %v", e.Code, e.msg)
}

var (
	// IllegalFunctionError with exception code 1, is returned when the
	// function received is not an allowable action fwith the slave.
	IllegalFunctionError = Error{Code: 1, msg: "illegal function"}

	// IllegalAddressError with exception code 2, is returned when the
	// address received is not an allowable address fwith the slave.
	IllegalAddressError = Error{Code: 2, msg: "illegal address"}

	// IllegalDataValueError with exception code 3, is returned if the
	// request contains an value that is not allowable fwith the slave.
	IllegalDataValueError = Error{Code: 3, msg: "illegal data value"}

	// SlaveDeviceFailureError with exception code 4, is returned when
	// the server isn't able to handle the request.
	SlaveDeviceFailureError = Error{Code: 4, msg: "slave device failure"}

	// AcknowledgeError with exception code 5, is returned when the
	// server has received the request successfully, but needs a long time
	// to process the request.
	AcknowledgeError = Error{Code: 5, msg: "acknowledge"}

	// SlaveDeviceBusyError with exception 6, is returned when master is
	// busy processing a long-running command.
	SlaveDeviceBusyError = Error{Code: 6, msg: "slave device busy"}

	// NegativeAcknowledgeError with exception code 7, is returned for an
	// unsuccessful programming request using function code 13 or 14.
	NegativeAcknowledgeError = Error{Code: 7, msg: "negative acknowledge"}

	// MemoryParityError with exception code 8 is returned to indicate that
	// the extended file area failed to pass a consistency check. May only
	// returned for requests with function code 20 or 21.
	MemoryParityError = Error{Code: 8, msg: "memory parity error"}

	// GatewayPathUnavailableError with exception code 10 indicates that
	// the gateway was unable to allocate an internal communication path
	// from the input port to the output port for processing the request.
	GatewayPathUnavailableError = Error{Code: 10, msg: "gateway path unavailable"}

	// GatewayTargetDeviceFailedToRespondError with exception code 11
	// indicates that the device is not present on the network.
	GatewayTargetDeviceFailedToRespondError = Error{Code: 11, msg: "gateway target device failed to respond"}
)

// Value is a value an integer ranging from range of -32768 through 65535.
type Value struct {
	v int
}

// NewValue creates a Value. It returns an error when given value is outside
// range of -32768 through 65535.
func NewValue(v int) (Value, error) {
	var value Value

	if err := value.Set(v); err != nil {
		return value, err
	}

	return value, nil
}

// Set sets the value. It returns an error when given value is outside range of
// -32768 through 65535.
func (v *Value) Set(value int) error {
	if value < -32768 || value > 65535 {
		return fmt.Errorf("%d doesn't fit in 16 bytes", v)
	}

	v.v = value
	return nil
}

// Get returns the value.
func (v Value) Get() int {
	return v.v
}

// MarshalBinary marshals a Value into byte slice with length of 2
// bytes.
func (v Value) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	var value interface{}
	if v.v < 0 {
		value = int16(v.v)
	} else {
		value = uint16(v.v)
	}

	if err := binary.Write(buf, binary.BigEndian, value); err != nil {
		return buf.Bytes(), fmt.Errorf("failed to marshal ResponseValue: %v", err)
	}

	return buf.Bytes(), nil
}

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

// UnmarshalBinary unmarshals a binary representation of MBAP.
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
	// Request is a Modbus request.
	MBAP

	FunctionCode uint8
	Data         []byte
}

// UnmarshalBinary unmarshals binary representation of Request.
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
	if r.FunctionCode == WriteSingleCoil || r.FunctionCode == WriteSingleRegister {
		resp.MBAP.Length = uint16(len(data) + 2)

	}

	return resp
}

// NewErrorResponse creates a error response.
func NewErrorResponse(r Request, err error) *Response {
	resp := &Response{
		MBAP:         r.MBAP,
		FunctionCode: r.FunctionCode + 0x80,
		exception:    true,
	}

	resp.Data = []byte{5}
	if err, ok := err.(Error); ok {
		resp.Data = []byte{err.Code}
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

	if !r.exception && r.FunctionCode != WriteSingleCoil && r.FunctionCode != WriteSingleRegister {
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
