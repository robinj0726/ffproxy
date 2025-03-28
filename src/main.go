package main

import (
	"fmt"
	"log"

	"github.com/robin/ffproxy/src/proxy"
)

func main() {
	fmt.Println("FFProxy starting...")
	
	// Create and start proxy server
	server := proxy.NewServer(":8080")
	log.Printf("Starting proxy server on %s", server.ListenAddr)
	
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
} 