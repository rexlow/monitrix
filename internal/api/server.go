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
	CurrentStatus      string          `json:"current_status"` // "online" or "offline"
	TotalChecks        int             `json:"total_checks"`
	OnlineChecks       int             `json:"online_checks"`
	OfflineChecks      int             `json:"offline_checks"`
	UptimePercentage   float64         `json:"uptime_percentage"`
	TotalDowntimeHours float64         `json:"total_downtime_hours"`
	DowntimeEvents     []DowntimeEvent `json:"downtime_events"`
	RecentDowntime     *DowntimeEvent  `json:"recent_downtime,omitempty"`
	TimeSinceLastCheck *time.Time      `json:"time_since_last_check,omitempty"`
}

// DowntimeEvent represents a period of internet connectivity loss
type DowntimeEvent struct {
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"` // nil if still ongoing
	Duration    int64      `json:"duration_seconds"`
	IsOngoing   bool       `json:"is_ongoing"`
	FailedHosts []string   `json:"failed_hosts"`
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
// Internet is considered DOWN only when ALL hosts fail to respond
func calculateStats(logs []storage.LogEntry) Stats {
	var downtimeEvents []DowntimeEvent
	var onlineChecks, offlineChecks int
	var totalDowntimeSeconds int64

	var lastStatus bool // true = online, false = offline
	var downtimeStart time.Time
	var downtimeFailedHosts []string
	var lastCheckTime *time.Time
	currentStatus := "online"

	statusInitialized := false

	for _, entry := range logs {
		// Check if ALL hosts failed (= internet is down)
		allFailed := true
		var failedHosts []string

		for _, result := range entry.Results {
			if result.Success {
				allFailed = false
			} else {
				failedHosts = append(failedHosts, result.Host)
			}
		}

		internetOnline := !allFailed
		lastCheckTime = &entry.Timestamp

		if internetOnline {
			onlineChecks++

			// Check if this ends a downtime period
			if statusInitialized && !lastStatus {
				endTime := entry.Timestamp
				duration := int64(endTime.Sub(downtimeStart).Seconds())
				totalDowntimeSeconds += duration

				downEvent := DowntimeEvent{
					StartTime:   downtimeStart,
					EndTime:     &endTime,
					Duration:    duration,
					IsOngoing:   false,
					FailedHosts: downtimeFailedHosts,
				}
				downtimeEvents = append(downtimeEvents, downEvent)
			}
			lastStatus = true
			currentStatus = "online"
		} else {
			offlineChecks++

			// Check if this starts a new downtime period
			if !statusInitialized || lastStatus {
				downtimeStart = entry.Timestamp
				downtimeFailedHosts = failedHosts
			}
			lastStatus = false
			currentStatus = "offline"
		}

		statusInitialized = true
	}

	// Handle ongoing downtime
	if statusInitialized && !lastStatus && lastCheckTime != nil {
		duration := int64(time.Since(downtimeStart).Seconds())
		downEvent := DowntimeEvent{
			StartTime:   downtimeStart,
			EndTime:     nil,
			Duration:    duration,
			IsOngoing:   true,
			FailedHosts: downtimeFailedHosts,
		}
		downtimeEvents = append(downtimeEvents, downEvent)
		totalDowntimeSeconds += duration
	}

	totalChecks := len(logs)
	uptimePercentage := 0.0
	if totalChecks > 0 {
		uptimePercentage = float64(onlineChecks) / float64(totalChecks) * 100
	}

	// Sort downtime events by start time (most recent first)
	for i := 0; i < len(downtimeEvents)/2; i++ {
		j := len(downtimeEvents) - 1 - i
		downtimeEvents[i], downtimeEvents[j] = downtimeEvents[j], downtimeEvents[i]
	}

	var recentDowntime *DowntimeEvent
	if len(downtimeEvents) > 0 {
		recentDowntime = &downtimeEvents[0]
	}

	return Stats{
		CurrentStatus:      currentStatus,
		TotalChecks:        totalChecks,
		OnlineChecks:       onlineChecks,
		OfflineChecks:      offlineChecks,
		UptimePercentage:   uptimePercentage,
		TotalDowntimeHours: float64(totalDowntimeSeconds) / 3600,
		DowntimeEvents:     downtimeEvents,
		RecentDowntime:     recentDowntime,
		TimeSinceLastCheck: lastCheckTime,
	}
}
