package monitor

import (
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

// Ping performs a TCP connection test to the host
func (m *Monitor) Ping(host string) PingResult {
	start := time.Now()
	result := PingResult{
		Host:      host,
		Timestamp: start,
	}

	// Try to establish TCP connection on port 80
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, "80"), m.timeout)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Latency = latency
		return result
	}

	conn.Close()
	result.Success = true
	result.Latency = latency

	return result
}

// PingAll pings all configured hosts
func (m *Monitor) PingAll() []PingResult {
	results := make([]PingResult, 0, len(m.hosts))
	for _, host := range m.hosts {
		result := m.Ping(host)
		results = append(results, result)
		fmt.Printf("[%s] %s: %v (latency: %dms)\n",
			result.Timestamp.Format("2006-01-02 15:04:05"),
			result.Host,
			result.Success,
			result.Latency)
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
