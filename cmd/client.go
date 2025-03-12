package cmd

import (
	"github.com/digitalrusher/tcp-tunnel/internal/tunnel"
	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start the TCP tunnel client",
	Run:   runClient,
}

func init() {
	rootCmd.AddCommand(clientCmd)

	clientCmd.Flags().StringP("server", "s", "localhost", "Server address")
	clientCmd.Flags().IntP("port", "p", 8080, "Server control port")
	clientCmd.Flags().IntP("service-port", "l", 80, "Local service port to forward")
}

func runClient(cmd *cobra.Command, args []string) {
	serverAddr, _ := cmd.Flags().GetString("server")
	serverPort, _ := cmd.Flags().GetInt("port")
	targetPort, _ := cmd.Flags().GetInt("service-port")

	client := tunnel.NewClient(serverAddr, serverPort, targetPort)
	client.Start()
}
