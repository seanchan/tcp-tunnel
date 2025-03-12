package tunnel

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type Client struct {
	serverAddr   string
	serverPort   int
	targetPort   int
	assignedPort int
	conn         net.Conn
	reader       *bufio.Reader
	logger       *zap.Logger
}

func NewClient(serverAddr string, serverPort, targetPort int) (*Client, error) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	return &Client{
		serverAddr: serverAddr,
		serverPort: serverPort,
		targetPort: targetPort,
		logger:     logger,
	}, nil
}

func (c *Client) Start() {
	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		if c.conn != nil {
			c.conn.Close()
		}
		os.Exit(0)
	}()

	for {
		err := c.connect()
		if err != nil {
			c.logger.Error("connection failed", zap.Error(err))
			time.Sleep(3 * time.Second)
			continue
		}

		if c.assignedPort != 0 {
			_, err = fmt.Fprintf(c.conn, "NEW %d\n", c.assignedPort)
		} else {
			_, err = fmt.Fprintf(c.conn, "NEW\n")
		}

		if err != nil {
			c.logger.Error("failed to send NEW command", zap.Error(err))
			c.conn.Close()
			time.Sleep(3 * time.Second)
			continue
		}

		heartbeatStop := make(chan struct{})
		go c.heartbeat(heartbeatStop)

		err = c.handleConnection()
		close(heartbeatStop)

		if err != nil {
			c.logger.Error("connection error", zap.Error(err))
			if c.conn != nil {
				c.conn.Close()
			}
			time.Sleep(3 * time.Second)
		}
	}
}

func (c *Client) connect() error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.serverAddr, c.serverPort))
	if err != nil {
		return err
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

func (c *Client) handleConnection() error {
	portStr, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read port: %v", err)
	}

	portStr = strings.TrimSpace(portStr)
	if strings.HasPrefix(portStr, "ERROR") {
		return fmt.Errorf("server error: %s", portStr)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port received: %v", err)
	}

	c.assignedPort = port
	fmt.Printf("Tunnel established: %s:%d -> localhost:%d\n",
		c.serverAddr, port, c.targetPort)

	for {
		cmd, err := c.reader.ReadString('\n')
		if err != nil {
			return err
		}

		cmd = strings.TrimSpace(cmd)
		if cmd == "CONNECT" {
			err = c.handleTunnelConnection()
			if err != nil {
				c.logger.Error("tunnel connection failed", zap.Error(err))
				continue
			}
		}
	}
}

func (c *Client) handleTunnelConnection() error {
	portMsg, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read port message: %v", err)
	}

	parts := strings.Fields(strings.TrimSpace(portMsg))
	if len(parts) != 2 || parts[0] != "PORT" {
		return fmt.Errorf("invalid port message: %s", portMsg)
	}

	tunnelPort, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid port number: %v", err)
	}

	c.logger.Info("attempting to connect to local service",
		zap.Int("local_port", c.targetPort))

	// Connect to local service
	localConn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", c.targetPort))
	if err != nil {
		return fmt.Errorf("failed to connect to local service: %v", err)
	}
	defer localConn.Close()

	c.logger.Info("connected to local service, establishing data channel",
		zap.Int("tunnel_port", tunnelPort))

	// Connect to server's data channel
	dataConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.serverAddr, tunnelPort))
	if err != nil {
		return fmt.Errorf("failed to connect to data channel: %v", err)
	}
	defer dataConn.Close()

	c.logger.Info("data channel established, starting data transfer")

	// Start bidirectional copy
	done := make(chan struct{})
	closeOnce := sync.Once{}

	go func() {
		n, err := io.Copy(dataConn, localConn)
		if err != nil && !isConnectionClosed(err) {
			c.logger.Error("error copying to server", zap.Error(err))
		}
		c.logger.Info("forward direction completed", zap.Int64("bytes", n))
		closeOnce.Do(func() { close(done) })
	}()

	go func() {
		n, err := io.Copy(localConn, dataConn)
		if err != nil && !isConnectionClosed(err) {
			c.logger.Error("error copying from server", zap.Error(err))
		}
		c.logger.Info("reverse direction completed", zap.Int64("bytes", n))
		closeOnce.Do(func() { close(done) })
	}()

	<-done
	c.logger.Info("data transfer completed")
	return nil
}

// 辅助函数：检查是否是正常的连接关闭
func isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "use of closed network connection") ||
		strings.Contains(err.Error(), "connection reset by peer") ||
		err == io.EOF
}

func (c *Client) heartbeat(stop chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, err := fmt.Fprintf(c.conn, "PING\n")
			if err != nil {
				return
			}

			c.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			response, err := c.reader.ReadString('\n')
			if err != nil {
				return
			}

			c.conn.SetReadDeadline(time.Time{})
			if strings.TrimSpace(response) != "PONG" {
				return
			}

		case <-stop:
			return
		}
	}
}
