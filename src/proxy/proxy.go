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

	// Check if the request is trying to proxy to itself
	if req.Host == s.ListenAddr || strings.HasPrefix(req.Host, "localhost") {
		log.Printf("Rejecting self-proxy request to %s", req.Host)
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
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

// getTargetAddress returns the target address with appropriate port
func getTargetAddress(host string) string {
	if !strings.Contains(host, ":") {
		// If no port specified, use default HTTP port
		return host + ":80"
	}
	return host
}

// handleHTTP handles regular HTTP requests
func (s *Server) handleHTTP(clientConn net.Conn, req *http.Request) {
	// Ensure the request URL is absolute
	if !req.URL.IsAbs() {
		req.URL.Scheme = "http"
		req.URL.Host = req.Host
	}

	// Get target address with port
	targetAddr := getTargetAddress(req.Host)
	log.Printf("Connecting to target: %s", targetAddr)

	// Create a connection to the target server
	targetConn, err := net.Dial("tcp", targetAddr)
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
	// Get target address with port (default to 443 for HTTPS)
	targetAddr := getTargetAddress(req.Host)
	if !strings.Contains(targetAddr, ":") {
		targetAddr = targetAddr + ":443"
	}
	log.Printf("Connecting to HTTPS target: %s", targetAddr)

	// Connect to the target server
	targetConn, err := net.Dial("tcp", targetAddr)
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
