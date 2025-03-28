package proxy

import (
	"log"
	"net"
)

// Server represents the proxy server
type Server struct {
	ListenAddr string
}

// NewServer creates a new proxy server instance
func NewServer(addr string) *Server {
	return &Server{
		ListenAddr: addr,
	}
}

// Start begins listening for connections
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("Listening on %s", s.ListenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

// handleConnection processes an individual client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	log.Printf("New connection from %s", conn.RemoteAddr().String())

	// Read the first byte to determine the protocol
	buf := make([]byte, 1)
	_, err := conn.Read(buf)
	if err != nil {
		log.Printf("Error reading from connection: %v", err)
		return
	}

	// TODO: Implement protocol-specific handling based on the first byte
	// For now, just echo back the received data
	conn.Write(buf)
}
