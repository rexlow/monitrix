package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"monitrix/internal/storage"
)

// Server handles HTTP API requests
type Server struct {
	dataDir string
	webDir  string
}

// NewServer creates a new API server
func NewServer(dataDir, webDir string) *Server {
	return &Server{
		dataDir: dataDir,
		webDir:  webDir,
	}
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/logs", s.handleLogs)
	http.HandleFunc("/api/stats", s.handleStats)

	fmt.Printf("Starting web dashboard at http://%s\n", addr)
	return http.ListenAndServe(addr, nil)
}

// handleIndex serves the dashboard HTML
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, s.webDir+"/index.html")
}

// handleLogs returns log entries with optional time filtering
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse query parameters for time range
	var startTime, endTime *time.Time

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = &t
		}
	}

	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = &t
		}
	}

	logs, err := storage.ReadLogs(s.dataDir, startTime, endTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read logs: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(logs)
}

// Stats represents aggregated statistics
type Stats struct {
	TotalChecks    int                 `json:"total_checks"`
	DowntimeEvents []DowntimeEvent     `json:"downtime_events"`
	HostStats      map[string]HostStat `json:"host_stats"`
}

// DowntimeEvent represents a period of downtime
type DowntimeEvent struct {
	Host      string    `json:"host"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  int64     `json:"duration_seconds"`
}

// HostStat represents statistics for a specific host
type HostStat struct {
	TotalPings   int     `json:"total_pings"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatency   float64 `json:"avg_latency_ms"`
}

// handleStats returns aggregated statistics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse query parameters for time range
	var startTime, endTime *time.Time

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = &t
		}
	}

	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = &t
		}
	}

	logs, err := storage.ReadLogs(s.dataDir, startTime, endTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read logs: %v", err), http.StatusInternalServerError)
		return
	}

	stats := calculateStats(logs)
	json.NewEncoder(w).Encode(stats)
}

// calculateStats computes statistics from log entries
func calculateStats(logs []storage.LogEntry) Stats {
	hostStats := make(map[string]HostStat)
	var downtimeEvents []DowntimeEvent

	// Track downtime periods
	lastState := make(map[string]bool)
	downtimeStart := make(map[string]time.Time)

	for _, entry := range logs {
		for _, result := range entry.Results {
			stat := hostStats[result.Host]
			stat.TotalPings++

			if result.Success {
				stat.SuccessCount++
				stat.AvgLatency = (stat.AvgLatency*float64(stat.SuccessCount-1) + float64(result.Latency)) / float64(stat.SuccessCount)

				// Check if this ends a downtime period
				if wasUp, exists := lastState[result.Host]; exists && !wasUp {
					downEvent := DowntimeEvent{
						Host:      result.Host,
						StartTime: downtimeStart[result.Host],
						EndTime:   result.Timestamp,
						Duration:  int64(result.Timestamp.Sub(downtimeStart[result.Host]).Seconds()),
					}
					downtimeEvents = append(downtimeEvents, downEvent)
				}
				lastState[result.Host] = true
			} else {
				stat.FailureCount++

				// Check if this starts a new downtime period
				if wasUp, exists := lastState[result.Host]; !exists || wasUp {
					downtimeStart[result.Host] = result.Timestamp
				}
				lastState[result.Host] = false
			}

			if stat.TotalPings > 0 {
				stat.SuccessRate = float64(stat.SuccessCount) / float64(stat.TotalPings) * 100
			}

			hostStats[result.Host] = stat
		}
	}

	return Stats{
		TotalChecks:    len(logs),
		DowntimeEvents: downtimeEvents,
		HostStats:      hostStats,
	}
}
