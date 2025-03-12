package tunnel

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	controlPort int
	minPort     int
	maxPort     int
	nextPort    int
	tunnels     map[int]*Tunnel
	mu          sync.Mutex
}

type Tunnel struct {
	clientConn net.Conn
	publicPort int
	targetPort int
}

func NewServer(controlPort, minPort, maxPort int) *Server {
	return &Server{
		controlPort: controlPort,
		minPort:     minPort,
		maxPort:     maxPort,
		nextPort:    minPort,
		tunnels:     make(map[int]*Tunnel),
	}
}

func (s *Server) Start() {
	// 监听控制端口，等待客户端连接
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.controlPort))
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	fmt.Printf("Server listening on control port %d\n", s.controlPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}
		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(clientConn net.Conn) {
	// 分配一个可用端口
	assignedPort := s.allocatePort()
	if assignedPort == -1 {
		fmt.Println("No available ports")
		clientConn.Close()
		return
	}

	// 创建新的隧道
	tunnel := &Tunnel{
		clientConn: clientConn,
		publicPort: assignedPort,
	}

	s.mu.Lock()
	s.tunnels[assignedPort] = tunnel
	s.mu.Unlock()

	// 启动公共端口监听
	go s.startTunnelListener(tunnel)

	// 通知客户端分配的端口
	fmt.Fprintf(clientConn, "%d\n", assignedPort)
	fmt.Printf("New tunnel established: %d\n", assignedPort)
}

func (s *Server) allocatePort() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	startPort := s.nextPort
	for {
		if s.nextPort > s.maxPort {
			s.nextPort = s.minPort
		}
		port := s.nextPort
		s.nextPort++

		if _, inUse := s.tunnels[port]; !inUse {
			return port
		}
		if s.nextPort == startPort {
			return -1 // 所有端口都在使用中
		}
	}
}

func (s *Server) startTunnelListener(tunnel *Tunnel) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", tunnel.publicPort))
	if err != nil {
		fmt.Printf("Failed to start tunnel listener on port %d: %v\n", tunnel.publicPort, err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection on tunnel %d: %v\n", tunnel.publicPort, err)
			continue
		}
		go s.handleTunnelConnection(conn, tunnel)
	}
}

func (s *Server) handleTunnelConnection(conn net.Conn, tunnel *Tunnel) {
	defer conn.Close()

	// 通知客户端新连接
	fmt.Fprintf(tunnel.clientConn, "CONNECT\n")

	// 等待客户端创建数据通道
	dataConn, err := s.waitForDataChannel(tunnel)
	if err != nil {
		fmt.Printf("Failed to establish data channel: %v\n", err)
		return
	}
	defer dataConn.Close()

	// 双向转发数据
	go func() {
		io.Copy(dataConn, conn)
	}()
	io.Copy(conn, dataConn)
}

func (s *Server) waitForDataChannel(tunnel *Tunnel) (net.Conn, error) {
	// 使用随机端口而不是控制端口
	listener, err := net.Listen("tcp", ":0") // 使用 :0 让系统分配随机可用端口
	if err != nil {
		return nil, err
	}
	defer listener.Close()

	// 获取实际分配的端口
	addr := listener.Addr().(*net.TCPAddr)

	// 将随机分配的端口号发送给客户端
	fmt.Fprintf(tunnel.clientConn, "PORT %d\n", addr.Port)

	// 设置接受连接超时
	listener.(*net.TCPListener).SetDeadline(time.Now().Add(10 * time.Second))

	conn, err := listener.Accept()
	if err != nil {
		return nil, err
	}

	return conn, nil
}
