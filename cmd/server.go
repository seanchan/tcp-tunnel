package cmd

import (
	"fmt"

	"github.com/digitalrusher/tcp-tunnel/internal/tunnel"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the TCP tunnel server",
	Run:   runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// 添加命令行参数
	serverCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	serverCmd.Flags().Int("min-port", 10000, "Minimum port number for tunnel connections")
	serverCmd.Flags().Int("max-port", 20000, "Maximum port number for tunnel connections")
}

func runServer(cmd *cobra.Command, args []string) {
	port, _ := cmd.Flags().GetInt("port")
	minPort, _ := cmd.Flags().GetInt("min-port")
	maxPort, _ := cmd.Flags().GetInt("max-port")

	fmt.Printf("Starting server on port %d (tunnel ports: %d-%d)\n",
		port, minPort, maxPort)
	server := tunnel.NewServer(port, minPort, maxPort)
	server.Start()
	// TODO: 在这里添加您的服务器启动逻辑
	// 例如：启动 TCP 监听器，处理连接等
}
