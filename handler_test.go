package modbus

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadHandler(t *testing.T) {
	h := NewReadHandler(func(unitID, start, quantity int) ([]int16, error) {
		assert.Equal(t, 0, unitID)
		assert.Equal(t, 5, start)
		assert.Equal(t, 3, quantity)

		return []int16{0, 1, 1}, nil
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
			[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x9, 0x0, 0x3, 0x6, 0x0, 0x1, 0x0, 0x1, 0x0, 0x0},
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
		input    []int16
		expected []int8
	}{
		{[]int16{0, 1, 1, 1}, []int8{0xe}},
		{[]int16{1, 0, 1, 0, 1, 0, 1, 0, 1}, []int8{0x1, 0x55}},
		{[]int16{1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0}, []int8{0x0, 0x1, 0x1}},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, reduce(test.input))
	}
}
