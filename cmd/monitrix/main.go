package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"monitrix/internal/api"
	"monitrix/internal/monitor"
	"monitrix/internal/storage"
)

// getEnv retrieves environment variable with fallback default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getHosts retrieves hosts from environment or returns defaults
func getHosts() []string {
	hostsEnv := os.Getenv("MONITOR_HOSTS")
	if hostsEnv != "" {
		hosts := strings.Split(hostsEnv, ",")
		// Trim whitespace from each host
		for i, host := range hosts {
			hosts[i] = strings.TrimSpace(host)
		}
		return hosts
	}
	// Default hosts - using reliable, geographically distributed services
	return []string{
		"1.1.1.1",        // Cloudflare DNS (very reliable)
		"8.8.8.8",        // Google DNS (very reliable)
		"google.com",     // Google (Americas)
		"cloudflare.com", // Cloudflare (Global CDN)
		"github.com",     // GitHub (Tech infrastructure)
	}
}

// getPingInterval retrieves ping interval from environment or returns default
func getPingInterval() time.Duration {
	intervalEnv := os.Getenv("MONITOR_INTERVAL")
	if intervalEnv != "" {
		if seconds, err := strconv.Atoi(intervalEnv); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	// Default to 30 seconds
	return 30 * time.Second
}

func main() {
	// Configuration with environment variable support
	hosts := getHosts()
	pingInterval := getPingInterval()
	pingTimeout := 5 * time.Second
	webAddr := getEnv("WEB_ADDR", "0.0.0.0:8080")

	// Get absolute paths
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
		os.Exit(1)
	}
	baseDir := filepath.Dir(execPath)
	dataDir := filepath.Join(baseDir, "..", "..", "data")
	webDir := filepath.Join(baseDir, "..", "..", "web")

	// For development, use current directory
	if wd, err := os.Getwd(); err == nil {
		dataDir = filepath.Join(wd, "data")
		webDir = filepath.Join(wd, "web")
	}

	fmt.Printf("Monitrix - Network Monitoring Tool\n")
	fmt.Printf("===================================\n")
	fmt.Printf("Monitoring hosts: %v\n", hosts)
	fmt.Printf("Check interval: %v\n", pingInterval)
	fmt.Printf("Data directory: %s\n", dataDir)
	fmt.Printf("Web directory: %s\n", webDir)
	fmt.Printf("\n")

	// Initialize storage
	fileStorage, err := storage.NewFileStorage(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}
	defer fileStorage.Close()

	// Initialize monitor
	mon := monitor.NewMonitor(hosts, pingInterval, pingTimeout)

	// Create channels for communication
	resultChan := make(chan []monitor.PingResult, 10)
	stopChan := make(chan struct{})

	// Start monitoring in background
	go mon.Start(resultChan, stopChan)

	// Start storage writer
	go func() {
		for results := range resultChan {
			if err := fileStorage.Save(results); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save results: %v\n", err)
			}
		}
	}()

	// Start web server in background
	server := api.NewServer(dataDir, webDir)
	go func() {
		if err := server.Start(webAddr); err != nil {
			panic(err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down gracefully...")
	close(stopChan)
	time.Sleep(1 * time.Second)
	close(resultChan)
}
