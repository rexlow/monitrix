package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"monitrix/internal/monitor"
)

// FileStorage handles storing ping results to file
type FileStorage struct {
	filePath string
	mu       sync.Mutex
	file     *os.File
}

// LogEntry represents a log entry in the file
type LogEntry struct {
	Timestamp time.Time            `json:"timestamp"`
	Results   []monitor.PingResult `json:"results"`
}

// NewFileStorage creates a new file storage instance
func NewFileStorage(dataDir string) (*FileStorage, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filename := fmt.Sprintf("network_monitor_%s.jsonl", time.Now().Format("2006-01-02"))
	filePath := filepath.Join(dataDir, filename)

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileStorage{
		filePath: filePath,
		file:     file,
	}, nil
}

// Save writes ping results to the log file
func (fs *FileStorage) Save(results []monitor.PingResult) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Results:   results,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	data = append(data, '\n')
	if _, err := fs.file.Write(data); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// Close closes the log file
func (fs *FileStorage) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.file != nil {
		return fs.file.Close()
	}
	return nil
}

// ReadLogs reads all log entries from files in the data directory
func ReadLogs(dataDir string, startTime, endTime *time.Time) ([]LogEntry, error) {
	files, err := filepath.Glob(filepath.Join(dataDir, "network_monitor_*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	var allEntries []LogEntry

	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", filePath, err)
			continue
		}

		decoder := json.NewDecoder(file)

		for decoder.More() {
			var entry LogEntry
			if err := decoder.Decode(&entry); err != nil {
				continue
			}

			// Filter by time range if specified
			if startTime != nil && entry.Timestamp.Before(*startTime) {
				continue
			}
			if endTime != nil && entry.Timestamp.After(*endTime) {
				continue
			}

			allEntries = append(allEntries, entry)
		}
		file.Close()
	}

	return allEntries, nil
}
