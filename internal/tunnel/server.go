package tunnel

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Server struct {
	controlPort int
	minPort     int
	maxPort     int
	nextPort    int
	tunnels     map[int]*Tunnel
	mu          sync.Mutex
	logger      *zap.Logger
}

type Tunnel struct {
	clientConn net.Conn
	publicPort int
	targetPort int
	listener   net.Listener
	active     bool
	mu         sync.Mutex
}

func NewServer(controlPort, minPort, maxPort int) (*Server, error) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	return &Server{
		controlPort: controlPort,
		minPort:     minPort,
		maxPort:     maxPort,
		nextPort:    minPort,
		tunnels:     make(map[int]*Tunnel),
		logger:      logger,
	}, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.controlPort))
	if err != nil {
		return err
	}

	s.logger.Info("server started", zap.Int("port", s.controlPort))

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	s.logger.Info("new connection established",
		zap.String("remote_addr", conn.RemoteAddr().String()))

	reader := bufio.NewReader(conn)
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return
	}

	command := strings.TrimSpace(firstLine)
	var tunnel *Tunnel

	if strings.HasPrefix(command, "NEW") {
		var requestedPort int
		parts := strings.Fields(command)
		if len(parts) > 1 {
			requestedPort, _ = strconv.Atoi(parts[1])
		}

		port := s.allocatePort(requestedPort)
		if port == -1 {
			fmt.Fprintf(conn, "ERROR no ports available\n")
			conn.Close()
			return
		}

		s.mu.Lock()
		if existingTunnel, exists := s.tunnels[port]; exists {
			existingTunnel.clientConn = conn
			existingTunnel.active = true
			tunnel = existingTunnel
		} else {
			tunnel = &Tunnel{
				publicPort: port,
				clientConn: conn,
				active:     true,
			}
			s.tunnels[port] = tunnel
			go s.startTunnelListener(tunnel)
		}
		s.mu.Unlock()

		fmt.Fprintf(conn, "%d\n", port)
		s.logger.Info("assigned port to client",
			zap.Int("port", port),
			zap.String("client", conn.RemoteAddr().String()))
	}

	// Start heartbeat
	heartbeatFailed := make(chan struct{})
	go s.handleHeartbeat(conn, reader, heartbeatFailed)

	<-heartbeatFailed
	if tunnel != nil {
		tunnel.active = false
		tunnel.clientConn = nil
	}
	conn.Close()
}

func (s *Server) startTunnelListener(tunnel *Tunnel) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", tunnel.publicPort))
	if err != nil {
		s.logger.Error("failed to start tunnel listener",
			zap.Int("port", tunnel.publicPort),
			zap.Error(err))
		return
	}
	tunnel.listener = listener
	s.logger.Info("started tunnel listener", zap.Int("public_port", tunnel.publicPort))

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return
		}
		go s.handleTunnelConnection(conn, tunnel)
	}
}

func (s *Server) handleTunnelConnection(conn net.Conn, tunnel *Tunnel) {
	defer conn.Close()
	s.logger.Info("new connection on tunnel",
		zap.String("remote_addr", conn.RemoteAddr().String()),
		zap.Int("public_port", tunnel.publicPort))

	tunnel.mu.Lock()
	if !tunnel.active || tunnel.clientConn == nil {
		tunnel.mu.Unlock()
		s.logger.Info("tunnel not active, rejecting connection")
		return
	}
	clientConn := tunnel.clientConn
	tunnel.mu.Unlock()

	// 设置连接超时
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Notify client of new connection
	_, err := fmt.Fprintf(clientConn, "CONNECT\n")
	if err != nil {
		s.logger.Error("failed to send CONNECT command", zap.Error(err))
		return
	}

	// Create temporary listener for data channel
	dataListener, err := net.Listen("tcp", ":0")
	if err != nil {
		s.logger.Error("failed to create data listener", zap.Error(err))
		return
	}
	defer dataListener.Close()

	dataPort := dataListener.Addr().(*net.TCPAddr).Port
	s.logger.Info("created data channel",
		zap.Int("data_port", dataPort))

	_, err = fmt.Fprintf(clientConn, "PORT %d\n", dataPort)
	if err != nil {
		s.logger.Error("failed to send PORT command", zap.Error(err))
		return
	}

	// Accept data connection from client with timeout
	dataListener.(*net.TCPListener).SetDeadline(time.Now().Add(5 * time.Second))
	dataConn, err := dataListener.Accept()
	if err != nil {
		s.logger.Error("failed to accept data connection", zap.Error(err))
		return
	}
	defer dataConn.Close()

	// 连接建立后清除超时设置
	conn.SetDeadline(time.Time{})
	dataConn.SetDeadline(time.Time{})

	s.logger.Info("data connection established")

	// Start bidirectional copy
	done := make(chan struct{})
	closeOnce := sync.Once{}

	go func() {
		n, err := io.Copy(conn, dataConn)
		if err != nil && !isConnectionClosed(err) {
			s.logger.Error("error copying data to client", zap.Error(err))
		}
		s.logger.Info("forward direction completed", zap.Int64("bytes", n))
		closeOnce.Do(func() { close(done) })
	}()

	go func() {
		n, err := io.Copy(dataConn, conn)
		if err != nil && !isConnectionClosed(err) {
			s.logger.Error("error copying data from client", zap.Error(err))
		}
		s.logger.Info("reverse direction completed", zap.Int64("bytes", n))
		closeOnce.Do(func() { close(done) })
	}()

	<-done
	s.logger.Info("tunnel connection completed")
}

func (s *Server) allocatePort(requestedPort int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果请求了特定端口且该端口可用
	if requestedPort >= s.minPort && requestedPort <= s.maxPort {
		if tunnel, exists := s.tunnels[requestedPort]; !exists || !tunnel.active {
			return requestedPort
		}
	}

	// 否则分配新端口
	for port := s.nextPort; port <= s.maxPort; port++ {
		if _, exists := s.tunnels[port]; !exists {
			s.nextPort = port + 1
			return port
		}
	}
	return -1
}

func (s *Server) handleHeartbeat(conn net.Conn, reader *bufio.Reader, failed chan struct{}) {
	defer close(failed)

	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		command := strings.TrimSpace(line)
		if command == "PING" {
			_, err := fmt.Fprintf(conn, "PONG\n")
			if err != nil {
				return
			}
		}
	}
}
