package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
)

// Signedness controls the signedness of values for Writehandler's. A value can
// be unsigned (capabale of representing only non-negative integers) or signed
// (capable of representing negative integers as well).
type Signedness int

const (
	// Unsigned set signedness to unsigned.
	Unsigned Signedness = iota

	// Signed sets signedness to signed.
	Signed
)

// A Handler responds to a Modbus request.
type Handler interface {
	ServeModbus(w io.Writer, r Request)
}

// ReadHandlerFunc is an adapter to allow the use of ordinary functions as
// handlers for Modbus read functions.
type ReadHandlerFunc func(unitID, start, quantity int) ([]Value, error)

// ReadHandler can be used to respond on Modbus request with function codes
// 1, 2, 3 and 4.
type ReadHandler struct {
	handle ReadHandlerFunc
}

// NewReadHandler creates a new ReadHandler.
func NewReadHandler(h ReadHandlerFunc) *ReadHandler {
	return &ReadHandler{
		handle: h,
	}
}

// ServeModbus writes a Modbus response.
func (h ReadHandler) ServeModbus(w io.Writer, req Request) {
	start := int(binary.BigEndian.Uint16(req.Data[:2]))
	quantity := int(binary.BigEndian.Uint16(req.Data[2:4]))

	values, err := h.handle(int(req.UnitID), start, quantity)
	if err != nil {
		respond(w, NewErrorResponse(req, err))
		return
	}

	var data []byte

	switch req.FunctionCode {
	case ReadCoils, ReadDiscreteInputs:
		data = append(data, reduce(values)...)
	default:
		for _, v := range values {
			b, err := v.MarshalBinary()
			if err != nil {
				respond(w, NewErrorResponse(req, SlaveDeviceFailureError))
				return
			}

			data = append(data, b...)
		}
	}

	respond(w, NewResponse(req, data))
}

func respond(w io.Writer, resp *Response) {
	data, err := resp.MarshalBinary()
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	if _, err := w.Write(data); err != nil {
		log.Printf("Failed to respond to client: %v", err)
	}
}

// reduce takes slice like [1, 0, 1, 0, 0, 1] and reduces that to a byte.
func reduce(values []Value) []byte {
	length := len(values) / 8
	if len(values)%8 > 0 {
		length++
	}
	reduced := make([]byte, length)

	n := length - 1

	// Iterate over 8 values a time.
	for i := 0; i <= len(values); i = i + 8 {
		end := i + 8
		if end > len(values)-1 {
			end = len(values)
		}

		b := values[i:end]
		for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
			b[i], b[j] = b[j], b[i]
		}

		for _, v := range b {
			reduced[n] = reduced[n] << 1
			if v.Get() > 0 {
				reduced[n] = reduced[n] | 1
			}
		}

		n--
	}

	return reduced
}

// WriteHandlerFunc is an adapter to allow the use of ordinary functions as
// handlers for Modbus write functions.
type WriteHandlerFunc func(unitID, start int, values []Value) error

// WriteHandler can be used to respond on Modbus request with function codes
// 5 and 6.
type WriteHandler struct {
	handler    WriteHandlerFunc
	signedness Signedness
}

// NewWriteHandler creates a new WriteHandler.
func NewWriteHandler(h WriteHandlerFunc, s Signedness) *WriteHandler {
	return &WriteHandler{
		handler:    h,
		signedness: s,
	}
}

// ServeModbus handles a Modbus request and returns a response.
func (h WriteHandler) ServeModbus(w io.Writer, req Request) {
	var err error
	var resp *Response
	var values []Value
	start := int(binary.BigEndian.Uint16(req.Data[:2]))

	switch req.FunctionCode {
	case WriteSingleCoil:
		values, err = h.handleWriteSingleCoil(req)
	case WriteSingleRegister:
		values, err = h.handleWriteSingleRegister(req)
	case WriteMultipleRegisters:
		values, err = h.handleWriteMultipleRegisters(req)
	}

	if err != nil {
		respond(w, NewErrorResponse(req, err))
		return
	}

	err = h.handler(int(req.UnitID), start, values)

	if err != nil {
		respond(w, NewErrorResponse(req, err))
		return
	}

	resp = NewResponse(req, req.Data[0:4])
	respond(w, resp)
}

func (h WriteHandler) handleWriteSingleCoil(req Request) ([]Value, error) {
	var v Value
	values := make([]Value, 1)
	if err := v.UnmarshalBinary(req.Data[2:4], Unsigned); err != nil {
		return values, fmt.Errorf("failed to hande write single coil request: %v", err)
	}

	if v.Get() != 0 {
		if err := v.Set(1); err != nil {
			return values, IllegalDataValueError
		}
	}
	values[0] = v

	return values, nil
}

func (h WriteHandler) handleWriteSingleRegister(req Request) ([]Value, error) {
	var v Value
	if err := v.UnmarshalBinary(req.Data[2:4], h.signedness); err != nil {
		return []Value{}, fmt.Errorf("failed to hande write single register request: %v", err)
	}
	return []Value{v}, nil
}

func (h WriteHandler) handleWriteMultipleRegisters(req Request) ([]Value, error) {
	quantity := int(binary.BigEndian.Uint16(req.Data[2:4]))
	values := []Value{}

	// The byte slice request.Data follows this format:
	//
	// ================ ===============
	// Field            Length (bytes)
	// ================ ===============
	// Starting Address 2
	// Quantity         2
	// Byte count       1
	// Values           n
	// ================ ===============
	//
	// The values are prepended with 5 bytes of meta data.
	// Every value is 2 bytes long.
	offset := 5
	if len(req.Data) != offset+(quantity*2) {
		return values, IllegalDataValueError
	}

	for i := 0; i < quantity*2; i += 2 {
		var v Value
		if err := v.UnmarshalBinary(req.Data[offset+i:offset+i+2], h.signedness); err != nil {
			return values, fmt.Errorf("failed to hande write multiple registers request: %v", err)
		}

		values = append(values, v)
	}

	return values, nil
}
