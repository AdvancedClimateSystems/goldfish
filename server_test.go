package modbus

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ErrorWriter is a writer that always fails to write.
type ErrorWriter struct{}

func (e ErrorWriter) Write([]byte) (int, error) { return 0, errors.New("") }

type RawHandler struct {
	handle func(w io.Writer, r Request)
}

func (h RawHandler) ServeModbus(w io.Writer, r Request) {
	h.handle(w, r)
}

func TestSetTimeout(t *testing.T) {
	s, err := NewServer(":")
	assert.Nil(t, err)
	assert.Equal(t, 0*time.Second, s.timeout)

	s.SetTimeout(5 * time.Second)
	assert.Equal(t, 5*time.Second, s.timeout)
}

// Connection is a struct implemention the io.ReadWriteCloser interface.
type Connection struct {
	read  func([]byte) (int, error)
	write func([]byte) (int, error)
	close func() error
}

func (c Connection) Read(b []byte) (int, error) { return c.read(b) }

func (c Connection) Write(b []byte) (int, error) { return c.write(b) }

func (c Connection) Close() error { return c.close() }

func TestListen(t *testing.T) {
	s := Server{}

	conn := Connection{
		read: func(b []byte) (int, error) {
			copy(b, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0})
			return 6, nil
		},

		write: func(b []byte) (int, error) {
			return 0, errors.New("")
		},
	}

	err := s.handleConn(conn)
	assert.NotNil(t, err)

	conn.read = func(b []byte) (int, error) {
		return 0, errors.New("")
	}

	err = s.handleConn(conn)
	assert.NotNil(t, err)
}

func TestReadMessage(t *testing.T) {
	s := Server{}

	tests := []struct {
		data []byte
	}{
		{[]byte{}},
		{[]byte{0x0}},
		{[]byte{0x0, 0x0}},
		{[]byte{0x0, 0x0, 0x0}},
		{[]byte{0x0, 0x0, 0x0, 0x0}},
		{[]byte{0x0, 0x0, 0x0, 0x0, 0x0}},
	}

	for _, test := range tests {
		_, err := s.readMessage(bufio.NewReader(bytes.NewReader(test.data)))
		assert.NotNil(t, err)
	}

	data := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0}
	msg, err := s.readMessage(bufio.NewReader(bytes.NewReader(data)))

	assert.Nil(t, err)
	assert.Equal(t, msg, data)
}

func TestExecuteAndRespond(t *testing.T) {
	s, _ := NewServer(":")
	writer := new(bytes.Buffer)
	req := &Request{FunctionCode: ReadCoils}

	// Try to execute a non-implemented function code. This fails,
	// therefore the server tries to send a IllegalFunction exception
	// response the the client. This should fail too because the writer
	// fails to write the response.
	err := s.executeAndRespond(ErrorWriter{}, req)
	assert.NotNil(t, err)

	// Again trying to execute a non-implemented function code. Now with
	// a function writer. This should succeed and the bytes making up a
	// IllegalFunction response should be written on the writer.
	err = s.executeAndRespond(writer, req)

	assert.Nil(t, err)
	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x3, 0x0, 0x81, 0x1}, writer.Bytes())

	// Try again, but now executing an implemented function code.
	// Everything should work.
	h := RawHandler{
		handle: func(w io.Writer, r Request) {
			assert.Equal(t, req, &r)
			assert.Equal(t, writer, w)
		},
	}

	s.Handle(ReadCoils, h)
	err = s.executeAndRespond(writer, req)
	assert.Nil(t, err)
}
