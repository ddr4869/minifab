package main

import (
	"log"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/orderer/server"
)

func init() {
	// Initialize logger with development config for CLI
	if err := logger.InitializeDevelopment(); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

}

func main() {
	if err := server.RootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute orderer command: %v", err)
	}
}
