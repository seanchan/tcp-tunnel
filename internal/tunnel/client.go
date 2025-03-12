package tunnel

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	serverAddr string
	serverPort int
	targetPort int
	conn       net.Conn
	reader     *bufio.Reader
}

func NewClient(serverAddr string, serverPort, targetPort int) *Client {
	return &Client{
		serverAddr: serverAddr,
		serverPort: serverPort,
		targetPort: targetPort,
	}
}

func (c *Client) Start() {
	for {
		err := c.connect()
		if err != nil {
			fmt.Printf("Failed to connect to server: %v\n", err)
			time.Sleep(5 * time.Second) // 重连延迟
			continue
		}

		err = c.handleConnection()
		if err != nil {
			fmt.Printf("Connection error: %v\n", err)
			if c.conn != nil {
				c.conn.Close()
			}
			time.Sleep(5 * time.Second) // 重连延迟
		}
	}
}

func (c *Client) connect() error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.serverAddr, c.serverPort))
	if err != nil {
		return err
	}
	c.conn = conn
	c.reader = bufio.NewReader(c.conn)
	return nil
}

func (c *Client) handleConnection() error {
	// 读取服务器分配的端口
	portStr, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read assigned port: %v", err)
	}

	assignedPort, err := strconv.Atoi(portStr[:len(portStr)-1])
	if err != nil {
		return fmt.Errorf("invalid port number: %v", err)
	}

	fmt.Printf("Tunnel established: %s:%d -> localhost:%d\n",
		c.serverAddr, assignedPort, c.targetPort)

	// 持续读取服务器的连接请求
	for {
		cmd, err := c.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read command: %v", err)
		}

		cmd = cmd[:len(cmd)-1] // 移除换行符
		if cmd == "CONNECT" {
			err = c.handleTunnelConnection()
			if err != nil {
				return err
			}
		}
	}
}

func (c *Client) handleTunnelConnection() error {
	// 连接本地服务
	localConn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", c.targetPort))
	if err != nil {
		fmt.Printf("Failed to connect to local service: %v\n", err)
		return err
	}
	defer localConn.Close()

	// 读取服务器分配的数据通道端口
	portMsg, err := c.reader.ReadString('\n')
	if err != nil {
		return err
	}
	portMsg = strings.TrimSpace(portMsg)
	var dataPort int
	if _, err := fmt.Sscanf(portMsg, "PORT %d", &dataPort); err != nil {
		return err
	}

	// 连接到服务器的数据通道端口
	dataConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.serverAddr, dataPort))
	if err != nil {
		fmt.Printf("Failed to create data channel: %v\n", err)
		return err
	}
	defer dataConn.Close()

	// 双向转发数据
	go func() {
		io.Copy(dataConn, localConn)
	}()
	io.Copy(localConn, dataConn)

	return nil
}
