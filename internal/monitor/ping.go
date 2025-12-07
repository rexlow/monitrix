package monitor

import (
	"context"
	"fmt"
	"net"
	"time"
)

// PingResult represents the result of a ping test
type PingResult struct {
	Host      string    `json:"host"`
	Success   bool      `json:"success"`
	Latency   int64     `json:"latency_ms"` // milliseconds
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Monitor handles network monitoring operations
type Monitor struct {
	hosts    []string
	interval time.Duration
	timeout  time.Duration
}

// NewMonitor creates a new monitor instance
func NewMonitor(hosts []string, interval, timeout time.Duration) *Monitor {
	return &Monitor{
		hosts:    hosts,
		interval: interval,
		timeout:  timeout,
	}
}

// Ping performs multiple connection tests to the host for reliability
// Connections taking more than 10 seconds are considered failures
func (m *Monitor) Ping(host string) PingResult {
	start := time.Now()
	result := PingResult{
		Host:      host,
		Timestamp: start,
	}

	// Maximum acceptable latency (10 seconds)
	maxAcceptableLatency := int64(10000) // 10 seconds in milliseconds

	// First, verify DNS resolution
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	resolver := &net.Resolver{}
	addrs, dnsErr := resolver.LookupHost(ctx, host)
	dnsLatency := time.Since(start).Milliseconds()

	if dnsErr != nil {
		result.Success = false
		result.Error = fmt.Sprintf("DNS lookup failed: %v", dnsErr)
		result.Latency = dnsLatency
		return result
	}

	if len(addrs) == 0 {
		result.Success = false
		result.Error = "No IP addresses found for host"
		result.Latency = dnsLatency
		return result
	}

	// Try multiple ports for better detection (HTTPS first, then HTTP)
	ports := []string{"443", "80"}
	var lastErr error

	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), m.timeout)
		latency := time.Since(start).Milliseconds()

		if err == nil {
			conn.Close()
			if latency > maxAcceptableLatency {
				result.Success = false
				result.Error = fmt.Sprintf("Connection too slow: %dms (max acceptable: %dms)", latency, maxAcceptableLatency)
				result.Latency = latency
				return result
			}

			result.Success = true
			result.Latency = latency
			return result
		}
		lastErr = err
	}

	// All ports failed
	latency := time.Since(start).Milliseconds()
	result.Success = false
	result.Error = lastErr.Error()
	result.Latency = latency

	return result
}

// PingAll pings all configured hosts and reports overall connectivity
func (m *Monitor) PingAll() []PingResult {
	results := make([]PingResult, 0, len(m.hosts))
	successCount := 0

	for _, host := range m.hosts {
		result := m.Ping(host)
		results = append(results, result)

		status := "✗ FAIL"
		errorMsg := ""
		if result.Success {
			status = "✓ OK"
			successCount++
		} else if result.Error != "" {
			errorMsg = fmt.Sprintf(" [%s]", result.Error)
			if len(errorMsg) > 60 {
				errorMsg = errorMsg[:57] + "...]"
			}
		}

		fmt.Printf("  %s %-20s (latency: %dms)%s\n",
			status,
			result.Host,
			result.Latency,
			errorMsg)
	}

	// Overall connectivity status
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	if successCount == 0 {
		fmt.Printf("\n[%s] ⚠️  INTERNET: OFFLINE - All hosts unreachable\n\n", timestamp)
	}

	return results
}

// Start begins continuous monitoring
func (m *Monitor) Start(resultChan chan<- []PingResult, stopChan <-chan struct{}) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Perform initial ping immediately
	results := m.PingAll()
	resultChan <- results

	for {
		select {
		case <-ticker.C:
			results := m.PingAll()
			resultChan <- results
		case <-stopChan:
			fmt.Println("Monitor stopped")
			return
		}
	}
}
