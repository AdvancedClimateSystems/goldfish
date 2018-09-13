package modbus

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

// Server is a Modbus server listens on a port and responds on incoming Modbus
// requests.
type Server struct {
	l        net.Listener
	handlers map[uint8]Handler
	timeout  time.Duration
	ErrorLog *log.Logger
}

// NewServer creates a new server on given address.
func NewServer(address string) (*Server, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to start Modbus server: %v", err)
	}

	return &Server{
		l:        l,
		timeout:  0,
		handlers: make(map[uint8]Handler),
	}, nil
}

// SetTimeout sets the timeout, which is the maximum duraion a request can take.
func (s *Server) SetTimeout(t time.Duration) {
	s.timeout = t
}

// Listen start listening for requests.
func (s *Server) Listen() {
	for {
		conn, err := s.l.Accept()
		if d := s.timeout; d != 0 {
			conn.SetReadDeadline(time.Now().Add(d))
		}

		if err != nil {
			s.logf("golfish: failed to accept incoming connection: %v", err)
			continue
		}

		go func() {
			if err := s.handleConn(conn); err != nil {
				s.logf("goldfish: unable to handle request from %v: %v", conn.RemoteAddr(), err)
			}

			if err := conn.Close(); err != nil {
				s.logf("goldfish: failed to close connection with %v: %v", conn.RemoteAddr(), err)
			}
		}()
	}
}

func (s *Server) handleConn(conn io.ReadWriteCloser) error {
	r := bufio.NewReader(conn)
	for {
		buf, err := s.readMessage(r)

		if err != nil {
			// An EOF error indicates the connection did not send new data. This
			// means the connection can be closed, but its not an error in the program.
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read message from connection: %v", err)
		}

		var req Request
		if err := req.UnmarshalBinary(buf); err != nil {
			return fmt.Errorf("failed to parse request: %v", err)
		}

		if err := s.executeAndRespond(conn, &req); err != nil {
			return fmt.Errorf("something went horribly wrong and server has to close connection: %v", err)
		}
	}
}

func (s *Server) readMessage(r *bufio.Reader) ([]byte, error) {
	b, err := r.Peek(6)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(b[4:6])

	buf := make([]byte, 6+length)
	_, err = r.Read(buf)

	if err != nil {
		return nil, fmt.Errorf("failed to read request: %v", err)
	}

	return buf, nil
}

func (s *Server) executeAndRespond(conn io.Writer, req *Request) error {
	h, ok := s.handlers[req.FunctionCode]
	if ok {
		h.ServeModbus(conn, *req)
		return nil
	}

	resp := NewErrorResponse(*req, IllegalFunctionError)
	data, err := resp.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to create response: %v", err)
	}

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to write response: %v", err)
	}

	return nil
}

// Handle registers the handler for the given function code.
func (s *Server) Handle(functionCode uint8, h Handler) {
	s.handlers[functionCode] = h
}

func (s *Server) logf(format string, args ...interface{}) {
	if s.ErrorLog != nil {
		s.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}
