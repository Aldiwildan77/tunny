package main

import (
	"log"
	"os"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Error executing command: %v\n", err)
		os.Exit(1)
	}
}
