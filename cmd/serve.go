package cmd

import (
	"errors"
	"fmt"
	"iwaradl/config"
	"iwaradl/server"

	"github.com/spf13/cobra"
)

var port int
var bindAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start iwara downloading daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		initRuntimeConfig()
		if port <= 0 || port > 65535 {
			return fmt.Errorf("invalid port: %d", port)
		}
		if config.Cfg.ApiToken == "" {
			return errors.New("api token is required in daemon mode, set --api-token or IWARADL_API_TOKEN")
		}
		fmt.Printf("Starting iwaradl daemon at %s:%d\n", bindAddr, port)
		return server.RunServer(bindAddr, port)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringVar(&bindAddr, "bind", "127.0.0.1", "listen address for daemon mode")
	serveCmd.Flags().IntVar(&port, "port", 23456, "listen port for daemon mode")
}
