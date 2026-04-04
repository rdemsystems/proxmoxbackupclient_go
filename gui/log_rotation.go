package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// MaxLogSize: Rotate when log reaches 10MB
	MaxLogSize = 10 * 1024 * 1024 // 10 MB

	// MaxLogFiles: Keep 5 rotated files (+ 1 current = 6 total)
	MaxLogFiles = 5
)

// RotatingLogger manages a log file with automatic rotation and compression
type RotatingLogger struct {
	path       string
	maxSize    int64
	maxFiles   int
	file       *os.File
	currentSize int64
	mu         sync.Mutex
}

// NewRotatingLogger creates a new rotating logger
func NewRotatingLogger(path string, maxSize int64, maxFiles int) (*RotatingLogger, error) {
	logger := &RotatingLogger{
		path:     path,
		maxSize:  maxSize,
		maxFiles: maxFiles,
	}

	// Get current file size if exists
	if info, err := os.Stat(path); err == nil {
		logger.currentSize = info.Size()
	}

	// Open log file (create if not exists)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	logger.file = file

	return logger, nil
}

// Write writes a log message and rotates if needed
func (l *RotatingLogger) Write(message string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if rotation is needed
	if l.currentSize >= l.maxSize {
		if err := l.rotate(); err != nil {
			return fmt.Errorf("failed to rotate log: %w", err)
		}
	}

	// Write message
	// CRITICAL: Check if file is nil (can happen if rotation failed)
	if l.file == nil {
		return fmt.Errorf("log file is nil, cannot write (rotation may have failed)")
	}
	n, err := l.file.WriteString(message)
	if err != nil {
		return fmt.Errorf("failed to write to log: %w", err)
	}

	// CRITICAL: Force immediate flush to disk to prevent data loss on crashes
	// This ensures log data is persisted even if process terminates unexpectedly
	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync log to disk: %w", err)
	}

	l.currentSize += int64(n)
	return nil
}

// rotate rotates the log file and compresses old logs
func (l *RotatingLogger) rotate() error {
	// Close current file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	// Generate timestamp for rotated file
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", l.path, timestamp)

	// Rename current log to rotated name
	if err := os.Rename(l.path, rotatedPath); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Compress the rotated file in background
	go func() {
		if err := compressLogFile(rotatedPath); err != nil {
			// Log to stderr if compression fails (can't use rotating logger here)
			fmt.Fprintf(os.Stderr, "Failed to compress log %s: %v\n", rotatedPath, err)
		}
	}()

	// Clean up old rotated logs (keep only maxFiles)
	go func() {
		if err := cleanupOldLogs(l.path, l.maxFiles); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to cleanup old logs: %v\n", err)
		}
	}()

	// Create new log file
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		// CRITICAL: If we can't create new log file, try to reopen the rotated file
		// to prevent losing all subsequent logs
		fmt.Fprintf(os.Stderr, "CRITICAL: Failed to create new log file %s: %v\n", l.path, err)
		fmt.Fprintf(os.Stderr, "Attempting to reopen rotated file %s\n", rotatedPath)

		// Try to reopen the rotated file as fallback
		fallbackFile, fallbackErr := os.OpenFile(rotatedPath, os.O_WRONLY|os.O_APPEND, 0600)
		if fallbackErr != nil {
			// Can't reopen rotated file either - this is FATAL
			fmt.Fprintf(os.Stderr, "FATAL: Cannot reopen rotated file: %v\n", fallbackErr)
			fmt.Fprintf(os.Stderr, "Logging is now BROKEN - backup may fail\n")
			// Set to nil to trigger fallback to stderr in writeLogToLogger
			l.file = nil
			return fmt.Errorf("failed to create new log file and reopen rotated file: %w", err)
		}

		// Successfully reopened rotated file - continue using it
		l.file = fallbackFile
		fmt.Fprintf(os.Stderr, "WARNING: Using rotated file %s for continued logging\n", rotatedPath)
		return nil
	}

	l.file = file
	l.currentSize = 0

	return nil
}

// Close closes the log file
func (l *RotatingLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// compressLogFile compresses a log file with gzip and removes the original
func compressLogFile(path string) error {
	// Open source file
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = src.Close() }()

	// Create compressed file
	gzPath := path + ".gz"
	dst, err := os.Create(gzPath)
	if err != nil {
		return fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer func() { _ = dst.Close() }()

	// Create gzip writer
	gzWriter := gzip.NewWriter(dst)
	defer func() { _ = gzWriter.Close() }()

	// Copy and compress
	if _, err := io.Copy(gzWriter, src); err != nil {
		return fmt.Errorf("failed to compress: %w", err)
	}

	// Close gzip writer to flush
	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Remove original file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	return nil
}

// cleanupOldLogs removes old rotated log files, keeping only maxFiles
func cleanupOldLogs(basePath string, maxFiles int) error {
	dir := filepath.Dir(basePath)
	baseName := filepath.Base(basePath)

	// List all rotated logs for this base name
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	// Filter rotated logs (both .gz and non-compressed)
	var rotatedLogs []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		// Match: basename.TIMESTAMP or basename.TIMESTAMP.gz
		if strings.HasPrefix(name, baseName+".") && name != baseName {
			rotatedLogs = append(rotatedLogs, entry)
		}
	}

	// If we have fewer files than maxFiles, nothing to clean
	if len(rotatedLogs) <= maxFiles {
		return nil
	}

	// Sort by name (timestamp is in the name, so this sorts by time)
	sort.Slice(rotatedLogs, func(i, j int) bool {
		return rotatedLogs[i].Name() < rotatedLogs[j].Name()
	})

	// Remove oldest files (keep only maxFiles newest)
	filesToRemove := len(rotatedLogs) - maxFiles
	for i := 0; i < filesToRemove; i++ {
		filePath := filepath.Join(dir, rotatedLogs[i].Name())
		if err := os.Remove(filePath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove old log %s: %v\n", filePath, err)
		}
	}

	return nil
}
