package cmd

import (
	"fmt"
	"iwaradl/server"

	"github.com/spf13/cobra"
)

var port int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start iwara downloading daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		initRuntimeConfig()
		if port <= 0 || port > 65535 {
			return fmt.Errorf("invalid port: %d", port)
		}
		return server.RunServer(port)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVar(&port, "port", 23456, "listen port for daemon mode")
}
