package main

import (
	"flag"
	"log"

	"github.com/digitalrusher/tcp-tunnel/internal/tunnel"
)

func main() {
	serverAddr := flag.String("server", "localhost", "Server address")
	serverPort := flag.Int("port", 8088, "Server control port")
	servicePort := flag.Int("service-port", 80, "Local service port to forward")
	flag.Parse()

	client, err := tunnel.NewClient(*serverAddr, *serverPort, *servicePort)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	client.Start()
}
