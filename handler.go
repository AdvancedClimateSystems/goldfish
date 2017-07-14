package modbus

import (
	"encoding/binary"
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
	start := int(binary.BigEndian.Uint16(req.Data[:2]))

	var data int
	if h.signedness == Unsigned {
		data = int(binary.BigEndian.Uint16(req.Data[2:4]))
	} else {
		data = int(int16(binary.BigEndian.Uint16(req.Data[2:4])))
	}

	v, err := NewValue(data)
	if err != nil {
		respond(w, NewErrorResponse(req, IllegalDataValueError))
		return
	}

	if req.FunctionCode == WriteSingleCoil {
		if v.Get() != 0 {
			if err := v.Set(1); err != nil {
				respond(w, NewErrorResponse(req, IllegalDataValueError))
				return
			}
		}
	}

	err = h.handler(int(req.UnitID), start, []Value{v})

	if err != nil {
		respond(w, NewErrorResponse(req, err))
		return
	}

	resp = NewResponse(req, req.Data)
	respond(w, resp)
}
