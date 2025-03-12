package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/digitalrusher/tcp-tunnel/internal/tunnel"
)

func main() {
	port := flag.Int("port", 8088, "Server control port")
	minPort := flag.Int("min-port", 10000, "Minimum port for tunnels")
	maxPort := flag.Int("max-port", 20000, "Maximum port for tunnels")
	flag.Parse()

	fmt.Printf("Starting server on port %d (tunnel ports: %d-%d)\n",
		*port, *minPort, *maxPort)

	server, err := tunnel.NewServer(*port, *minPort, *maxPort)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
