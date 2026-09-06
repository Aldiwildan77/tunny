package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "tunny",
	Short: "Location-aware network tunneling",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&configPath,
		"config",
		"c",
		"config.yaml",
		"Path to config file",
	)

	rootCmd.AddCommand(providerCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(tunnelCmd)
}
