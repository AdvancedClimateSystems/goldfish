package modbus

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
)

// A Handler responds to a Modbus request.
type Handler interface {
	ServeModbus(w io.Writer, r Request)
}

// ReadHandler can be used to respond on Modbus request with function codes
// 1,2, 3 and 4.
type ReadHandler struct {
	handle func(UnitID, start, quantity int) ([]int16, error)
}

// NewReadHandler creates a new ReadHandler.
func NewReadHandler(h func(unitID, start, quantity int) ([]int16, error)) *ReadHandler {
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

	buf := new(bytes.Buffer)

	var data []interface{}

	switch req.FunctionCode {
	case ReadCoils, ReadDiscreteInputs:
		for _, v := range reduce(values) {
			data = append(data, v)
		}
	default:
		for i := len(values) - 1; i >= 0; i-- {
			data = append(data, values[i])

		}
	}

	for _, v := range data {
		if err := binary.Write(buf, binary.BigEndian, v); err != nil {
			respond(w, NewErrorResponse(req, SlaveDeviceFailureError))
			return
		}
	}

	respond(w, NewResponse(req, buf.Bytes()))
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
func reduce(values []int16) []int8 {
	length := len(values) / 8
	if len(values)%8 > 0 {
		length++
	}
	reduced := make([]int8, length)

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
			if v > 0 {
				reduced[n] = reduced[n] | 1
			}
		}

		n--
	}

	return reduced
}

// WriteHandler can be used to respond on Modbus request with function codes
// 5 and 6.
type WriteHandler struct {
	handler func(unitID, start int, values []int16) error
}

// NewWriteHandler creates a new WriteHandler.
func NewWriteHandler(h func(unitID, start int, values []int16) error) *WriteHandler {
	return &WriteHandler{
		handler: h,
	}
}

// ServeModbus writes a Modbus response.
func (h WriteHandler) ServeModbus(w io.Writer, req Request) {
	var err error
	var resp *Response
	start := int(binary.BigEndian.Uint16(req.Data[:2]))

	switch req.FunctionCode {
	case WriteSingleCoil:
		v := []int16{int16(binary.BigEndian.Uint16(req.Data[2:4]))}
		if v[0] != 0 {
			v[0] = 1
		}
		err = h.handler(int(req.UnitID), start, v)
		if err == nil {
			resp = NewResponse(req, req.Data)
		}
	case WriteSingleRegister:
		v := []int16{int16(binary.BigEndian.Uint16(req.Data[2:4]))}
		err = h.handler(int(req.UnitID), start, v)
		if err == nil {
			resp = NewResponse(req, req.Data)
		}
	}

	if err != nil {
		respond(w, NewErrorResponse(req, err))
		return
	}

	respond(w, resp)
}
