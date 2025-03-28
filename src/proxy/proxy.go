package proxy

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
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

	log.Printf("HTTP Proxy listening on %s", s.ListenAddr)

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
func (s *Server) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	log.Printf("New connection from %s", clientConn.RemoteAddr().String())

	reader := bufio.NewReader(clientConn)

	// Read the HTTP request
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Error reading request: %v", err)
		return
	}

	log.Printf("Received request: %s %s %s", req.Method, req.Host, req.URL.String())

	// Handle CONNECT method differently (for HTTPS)
	if req.Method == http.MethodConnect {
		s.handleHTTPS(clientConn, req)
		return
	}

	// For regular HTTP requests
	s.handleHTTP(clientConn, req)
}

// handleHTTP handles regular HTTP requests
func (s *Server) handleHTTP(clientConn net.Conn, req *http.Request) {
	// Ensure the request URL is absolute
	if !req.URL.IsAbs() {
		req.URL.Scheme = "http"
		req.URL.Host = req.Host
	}

	// Create a connection to the target server
	targetConn, err := net.Dial("tcp", req.Host)
	if err != nil {
		log.Printf("Error connecting to target: %v", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer targetConn.Close()

	// Forward the modified request to the target
	err = req.Write(targetConn)
	if err != nil {
		log.Printf("Error writing to target: %v", err)
		return
	}

	// Copy the response back to the client
	_, err = io.Copy(clientConn, targetConn)
	if err != nil {
		log.Printf("Error copying response: %v", err)
	}
}

// handleHTTPS handles HTTPS CONNECT requests
func (s *Server) handleHTTPS(clientConn net.Conn, req *http.Request) {
	// Connect to the target server
	targetConn, err := net.Dial("tcp", req.Host)
	if err != nil {
		log.Printf("Error connecting to target: %v", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer targetConn.Close()

	// Send 200 OK to the client to indicate tunnel established
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	if err != nil {
		log.Printf("Error writing CONNECT response: %v", err)
		return
	}

	// Start bidirectional copy
	go func() {
		_, err := io.Copy(targetConn, clientConn)
		if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			log.Printf("Error copying to target: %v", err)
		}
	}()

	_, err = io.Copy(clientConn, targetConn)
	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		log.Printf("Error copying to client: %v", err)
	}
}
