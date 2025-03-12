package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tcp-tunnel",
	Short: "TCP tunnel for NAT traversal",
	Long: `TCP tunnel is a tool that helps you expose local services through NAT.
It consists of a server and a client component.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
