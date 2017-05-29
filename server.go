package modbus

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

// Server is a Modbus server listens on a port and responds on incoming Modbus
// requests.
type Server struct {
	l        net.Listener
	handlers map[uint8]Handler
}

// NewServer creates a new server on given address.
func NewServer(address string) (*Server, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to start Modbus server: %v", err)
	}

	return &Server{
		l:        l,
		handlers: make(map[uint8]Handler),
	}, nil
}

// Listen start listening for requests.
func (s *Server) Listen() {
	for {
		conn, err := s.l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Failed to  close connection with client: %v", err)
		}
	}()
	r := bufio.NewReader(conn)

	b, err := r.Peek(6)
	length := binary.BigEndian.Uint16(b[4:6])

	buf := make([]byte, 6+length)
	_, err = r.Read(buf)

	if err != nil {
		return
	}

	var req Request
	if err := req.UnmarshalBinary(buf); err != nil {
		log.Printf("Failed to parse request: %v", err)
		return
	}

	h, ok := s.handlers[req.FunctionCode]
	if ok {
		h.ServeModbus(conn, req)
		return
	}

	resp := NewErrorResponse(req, IllegalFunctionError)
	data, err := resp.MarshalBinary()
	if err != nil {
		panic(err)
	}

	if _, err := conn.Write(data); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

// Handle registers the handler for the given function code.
func (s *Server) Handle(functionCode uint8, h Handler) {
	s.handlers[functionCode] = h
}
