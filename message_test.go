package modbus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMBAP(t *testing.T) {
	tests := []struct {
		mbap MBAP
		data []byte
	}{
		{MBAP{TransactionID: 18, ProtocolID: 1, Length: 18, UnitID: 2}, []byte{0x0, 0x12, 0x0, 0x1, 0x0, 0x12, 0x2}},
		{MBAP{TransactionID: 18493, ProtocolID: 1, Length: 300, UnitID: 25}, []byte{0x48, 0x3d, 0x0, 0x1, 0x1, 0x2c, 0x19}},
		{MBAP{TransactionID: 54602, ProtocolID: 1, Length: 20110, UnitID: 91}, []byte{0xd5, 0x4a, 0x0, 0x1, 0x4e, 0x8e, 0x5b}},
	}

	for _, test := range tests {
		var m MBAP
		assert.Nil(t, m.UnmarshalBinary(test.data))
		assert.Equal(t, test.mbap, m)

		data, err := m.MarshalBinary()
		assert.Nil(t, err)
		assert.Equal(t, test.data, data)
	}

	var m MBAP

	// UnmarshalBinary should return error if length of given byte slice is not 7.
	assert.NotNil(t, m.UnmarshalBinary([]byte{}))
}

func TestRequest(t *testing.T) {
	mbap := MBAP{
		TransactionID: 1,
		ProtocolID:    1,
		Length:        6,
		UnitID:        3,
	}

	tests := []struct {
		request Request
		data    []byte
	}{
		// Read 5 coils starting from address 2.
		{Request{MBAP: mbap, FunctionCode: 1, Data: []byte{0x0, 0x2, 0x0, 0x5}}, []byte{0x0, 0x1, 0x0, 0x1, 0x0, 0x06, 0x3, 0x1, 0x0, 0x2, 0x0, 0x5}},
		// Read 2 read discrete inputs starting from address 19.
		{Request{MBAP: mbap, FunctionCode: 2, Data: []byte{0x0, 0x14, 0x0, 0x2}}, []byte{0x0, 0x1, 0x0, 0x1, 0x0, 0x06, 0x3, 0x2, 0x0, 0x14, 0x0, 0x2}},
	}

	for _, test := range tests {
		var r Request
		assert.Nil(t, r.UnmarshalBinary(test.data))
		assert.Equal(t, test.request, r)
	}
}

func TestResponse(t *testing.T) {
	request := Request{
		MBAP: MBAP{
			TransactionID: 1,
			ProtocolID:    1,
			Length:        5,
			UnitID:        3,
		},
		FunctionCode: 4,
		Data:         []byte{},
	}

	tests := []struct {
		response *Response
		data     []byte
	}{
		{NewErrorResponse(request, IllegalFunctionError), []byte{0x0, 0x1, 0x0, 0x1, 0x0, 0x03, 0x3, 0x84, 0x1}},
		{NewErrorResponse(request, AcknowledgeError), []byte{0x0, 0x1, 0x0, 0x1, 0x0, 0x03, 0x3, 0x84, 0x5}},
		{NewResponse(request, []byte{0x24, 0x41}), []byte{0x0, 0x1, 0x0, 0x1, 0x0, 0x05, 0x3, 0x4, 0x2, 0x24, 0x41}},
		{NewResponse(request, []byte{0x1, 0x9, 0x12, 0x3}), []byte{0x0, 0x1, 0x0, 0x1, 0x0, 0x07, 0x3, 0x4, 0x4, 0x1, 0x9, 0x12, 0x3}},
	}

	for _, test := range tests {
		data, err := test.response.MarshalBinary()
		assert.Nil(t, err)
		assert.Equal(t, test.data, data)
	}
}
