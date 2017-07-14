package modbus

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadHandler(t *testing.T) {
	h := NewReadHandler(func(unitID, start, quantity int) ([]Value, error) {
		assert.Equal(t, 0, unitID)
		assert.Equal(t, 5, start)
		assert.Equal(t, 3, quantity)

		return []Value{Value{0}, Value{1}, Value{1}}, nil
	})

	tests := []struct {
		req      Request
		expected []byte
	}{
		{
			Request{MBAP{}, ReadCoils, []byte{0x0, 0x5, 0x0, 0x3}},
			[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x1, 0x1, 0x6},
		},
		{
			Request{MBAP{}, ReadHoldingRegisters, []byte{0x0, 0x5, 0x0, 0x3}},
			[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x9, 0x0, 0x3, 0x6, 0x0, 0x0, 0x0, 0x1, 0x0, 0x1},
		},
	}

	for _, test := range tests {
		buf := new(bytes.Buffer)
		h.ServeModbus(buf, test.req)
		assert.Equal(t, test.expected, buf.Bytes())
	}
}

func TestReduce(t *testing.T) {
	tests := []struct {
		input    []Value
		expected []byte
	}{
		{[]Value{Value{0}, Value{1}, Value{1}, Value{1}}, []byte{0xe}},
		{[]Value{Value{1}, Value{0}, Value{1}, Value{0}, Value{1}, Value{0}, Value{1}, Value{0}, Value{1}}, []byte{0x1, 0x55}},
		{[]Value{Value{1}, Value{0}, Value{0}, Value{0}, Value{0}, Value{0}, Value{0}, Value{0}, Value{1}, Value{0}, Value{0}, Value{0}, Value{0}, Value{0}, Value{0}, Value{0}, Value{0}}, []byte{0x0, 0x1, 0x1}},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, reduce(test.input))
	}
}

func newWriteHandler(t *testing.T, unitID, start int, values []Value, response error, s Signedness) *WriteHandler {
	return NewWriteHandler(func(u, s int, v []Value) error {
		assert.Equal(t, unitID, u)
		assert.Equal(t, start, s)
		assert.Equal(t, values, v)

		return response
	}, s)
}

func TestWriteHandler(t *testing.T) {
	tests := []struct {
		req      Request
		h        *WriteHandler
		expected []byte
	}{
		{
			Request{MBAP{}, WriteSingleCoil, []byte{0x0, 0x1, 0x0, 0x0}},
			newWriteHandler(t, 0, 1, []Value{Value{0}}, nil, Signed),
			[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x0, 0x5, 0x0, 0x01, 0x0, 0x0},
		},
		{
			Request{MBAP{}, WriteSingleCoil, []byte{0x0, 0x1, 0xc, 0x1}},
			newWriteHandler(t, 0, 1, []Value{Value{1}}, IllegalFunctionError, Signed),
			[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x3, 0x0, 0x85, 0x01},
		},
		{
			Request{MBAP{}, WriteSingleRegister, []byte{0x0, 0x1, 0xf3, 0x88}},
			newWriteHandler(t, 0, 1, []Value{Value{-3192}}, nil, Signed),
			[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x0, 0x6, 0x0, 0x01, 0xf3, 0x88},
		},
		{
			Request{MBAP{}, WriteSingleRegister, []byte{0x0, 0x1, 0xf3, 0x88}},
			newWriteHandler(t, 0, 1, []Value{Value{62344}}, nil, Unsigned),
			[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x0, 0x6, 0x0, 0x01, 0xf3, 0x88},
		},
		{
			Request{MBAP{}, WriteSingleRegister, []byte{0x0, 0x1, 0xc, 0x78}},
			newWriteHandler(t, 0, 1, []Value{Value{3192}}, SlaveDeviceBusyError, Signed),
			[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x3, 0x0, 0x86, 0x6},
		},
	}

	for _, test := range tests {
		buf := new(bytes.Buffer)
		test.h.ServeModbus(buf, test.req)
		assert.Equal(t, test.expected, buf.Bytes())
	}
}
