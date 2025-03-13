package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
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
	logMutex     sync.Mutex
	logText      *widget.Label
}

func NewClient(serverAddr string, serverPort, targetPort int) (*Client, error) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	return &Client{
		serverAddr: serverAddr,
		serverPort: serverPort,
		targetPort: targetPort,
		logger:     logger,
		logText:    widget.NewLabel(""),
	}, nil
}

func (c *Client) Start() {
	go c.connect()
}

func (c *Client) connect() {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.serverAddr, c.serverPort))
	if err != nil {
		c.log("Connection failed: " + err.Error())
		return
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)

	c.log("Connected to server")

	// Send NEW command
	if c.assignedPort != 0 {
		_, err = fmt.Fprintf(c.conn, "NEW %d\n", c.assignedPort)
	} else {
		_, err = fmt.Fprintf(c.conn, "NEW\n")
	}

	if err != nil {
		c.log("Failed to send NEW command: " + err.Error())
		c.conn.Close()
		return
	}

	c.handleConnection()
}

func (c *Client) handleConnection() {
	portStr, err := c.reader.ReadString('\n')
	if err != nil {
		c.log("Failed to read port: " + err.Error())
		return
	}

	portStr = strings.TrimSpace(portStr)
	if strings.HasPrefix(portStr, "ERROR") {
		c.log("Server error: " + portStr)
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		c.log("Invalid port received: " + err.Error())
		return
	}

	c.assignedPort = port
	c.log(fmt.Sprintf("Tunnel established: %s:%d -> localhost:%d", c.serverAddr, port, c.targetPort))

	// Handle further commands...
}

func (c *Client) log(message string) {
	c.logMutex.Lock()
	defer c.logMutex.Unlock()
	c.logText.SetText(c.logText.Text + message + "\n")
}

func main() {
	a := app.New()
	w := a.NewWindow("TCP Tunnel Client")

	serverEntry := widget.NewEntry()
	serverEntry.SetPlaceHolder("Server Address")

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("Local Service Port")

	var client *Client // Declare the client variable

	connectButton := widget.NewButton("Connect", func() {
		serverAddr := serverEntry.Text
		serverPort := 8088 // Default port, can be made configurable
		targetPort, err := strconv.Atoi(portEntry.Text)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Invalid port: %s", portEntry.Text), w)
			return
		}

		client, err = NewClient(serverAddr, serverPort, targetPort)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		client.Start()
	})

	content := container.NewVBox(
		widget.NewLabel("Enter Server Address:"),
		serverEntry,
		widget.NewLabel("Enter Local Service Port:"),
		portEntry,
		connectButton,
		widget.NewLabel("Logs:"),
		client.logText,
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(400, 300))
	w.ShowAndRun()
}
