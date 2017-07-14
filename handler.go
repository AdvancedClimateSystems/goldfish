package modbus

import (
	"encoding/binary"
	"io"
	"log"
)

// A Handler responds to a Modbus request.
type Handler interface {
	ServeModbus(w io.Writer, r Request)
}

// ReadHandlerFunc is an adapter to allow the use of ordinary functions as
// handlers for Modbus read functions.
type ReadHandlerFunc func(unitID, start, quantity int) ([]Value, error)

// ReadHandler can be used to respond on Modbus request with function codes
// 1,2, 3 and 4.
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
	handler WriteHandlerFunc
}

// NewWriteHandler creates a new WriteHandler.
func NewWriteHandler(h WriteHandlerFunc) *WriteHandler {
	return &WriteHandler{
		handler: h,
	}
}

// ServeModbus writes a Modbus response.
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
	v, err := NewValue(int(binary.BigEndian.Uint16(req.Data[2:4])))
	values := []Value{v}

	if err != nil {
		return values, IllegalDataValueError
	}
	if v.Get() != 0 {
		if err := v.Set(1); err != nil {
			return values, IllegalDataValueError
		}
	}

	return values, nil
}

func (h WriteHandler) handleWriteSingleRegister(req Request) ([]Value, error) {
	v, err := NewValue(int(binary.BigEndian.Uint16(req.Data[2:4])))
	values := []Value{v}

	if err != nil {
		return values, IllegalDataValueError
	}

	return values, nil
}

func (h WriteHandler) handleWriteMultipleRegisters(req Request) ([]Value, error) {
	quantity := int(binary.BigEndian.Uint16(req.Data[2:4]))
	values := []Value{}

	offset := 5
	if len(req.Data) != offset+(quantity*2) {
		return values, IllegalDataValueError
	}

	for i := 0; i <= quantity; i += 2 {
		v, err := NewValue(int(binary.BigEndian.Uint16(req.Data[offset+i : offset+i+2])))
		if err != nil {
			return values, IllegalDataValueError
		}

		values = append(values, v)
	}

	return values, nil
}
