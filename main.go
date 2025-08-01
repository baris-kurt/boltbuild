package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Version information
const (
	Version = "1.0.0"
)

// Global configuration
var globalConfig *Config

func main() {
	// Simple argument parsing
	if len(os.Args) < 2 {
		fmt.Println("Usage: boltbuild [server|client] [config.yaml]")
		fmt.Println("  server - Start build server")
		fmt.Println("  client - Start build client with web interface")
		fmt.Println("  config.yaml - Optional path to configuration file (default: config.yaml)")
		os.Exit(1)
	}

	// Load configuration
	configPath := "config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	var err error
	globalConfig, err = LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger with config
	InitializeLogger(globalConfig)
	LogInfof("Configuration loaded from %s", configPath)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	mode := os.Args[1]
	switch mode {
	case "server":
		runServer(sigChan)
	case "client":
		runClient(sigChan)
	default:
		fmt.Printf("Invalid mode: %s. Use 'server' or 'client'\n", mode)
		os.Exit(1)
	}
}

// runServer starts a build server that accepts client connections
func runServer(sigChan chan os.Signal) {
	LogInfo("Starting BoltBuild - Server Mode")
	LogInfof("Build server will listen on port %d with capacity %d", globalConfig.Server.Port, globalConfig.Server.Capacity)

	// Create server (build worker)
	server := NewServer(globalConfig.Server.Port, globalConfig.Server.Capacity)

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			LogFatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	LogInfo("Shutting down server...")
}

// runClient starts a client with web interface that discovers and connects to servers
func runClient(sigChan chan os.Signal) {
	LogInfo("Starting BoltBuild - Client Mode")

	// Create client (build coordinator)
	client := NewClient()

	// Create web server
	webServer := NewWebServer(client, globalConfig.Web.Port)

	// Start web server in goroutine
	go func() {
		LogInfof("Web interface available at http://localhost:%d", globalConfig.Web.Port)
		if err := webServer.Start(); err != nil {
			LogFatalf("Web server failed: %v", err)
		}
	}()

	// Start client in goroutine
	go func() {
		if err := client.Start(); err != nil {
			LogFatalf("Client failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	LogInfo("Shutting down client...")
}
