package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
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
	path        string
	maxSize     int64
	maxFiles    int
	file        *os.File
	currentSize int64
	mu          sync.Mutex
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
			// Rotation failed but logger is still usable (truncate strategy)
			fmt.Fprintf(os.Stderr, "Log rotation failed: %v (continuing with current file)\n", err)
		}
	}

	if l.file == nil {
		return fmt.Errorf("log file is nil, cannot write")
	}
	n, err := l.file.WriteString(message)
	if err != nil {
		return fmt.Errorf("failed to write to log: %w", err)
	}

	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync log to disk: %w", err)
	}

	l.currentSize += int64(n)
	return nil
}

// rotate rotates the log file.
// On Windows, files must be closed before rename, so we use a copy+truncate
// strategy to avoid losing the file handle if rename fails.
func (l *RotatingLogger) rotate() error {
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", l.path, timestamp)

	if runtime.GOOS == "windows" {
		return l.rotateWindows(rotatedPath)
	}
	return l.rotatePosix(rotatedPath)
}

// rotatePosix uses rename (safe on Linux/Mac where open files can be renamed)
func (l *RotatingLogger) rotatePosix(rotatedPath string) error {
	// Close, rename, reopen
	_ = l.file.Close()

	if err := os.Rename(l.path, rotatedPath); err != nil {
		// Reopen the original file to keep logging
		l.reopenOrDie()
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Start background compression
	go l.compressAndCleanup(rotatedPath)

	// Open new log file
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		l.reopenOrDie()
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	l.file = file
	l.currentSize = 0
	return nil
}

// rotateWindows uses copy+truncate to avoid Windows file locking issues.
// The current file is copied to the rotated path, then truncated in place.
// This avoids Close→Rename→Open which breaks if Rename fails on Windows.
func (l *RotatingLogger) rotateWindows(rotatedPath string) error {
	// Sync before copying
	_ = l.file.Sync()

	// Seek to beginning for copying
	if _, err := l.file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek log file: %w", err)
	}

	// Copy current content to rotated file
	dst, err := os.Create(rotatedPath)
	if err != nil {
		// Seek back to end for continued writing
		_, _ = l.file.Seek(0, 2)
		return fmt.Errorf("failed to create rotated file: %w", err)
	}

	if _, err := io.Copy(dst, l.file); err != nil {
		_ = dst.Close()
		_ = os.Remove(rotatedPath)
		_, _ = l.file.Seek(0, 2)
		return fmt.Errorf("failed to copy log content: %w", err)
	}
	_ = dst.Close()

	// Truncate the original file (keep the handle open)
	if err := l.file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate log file: %w", err)
	}
	_, _ = l.file.Seek(0, 0)
	l.currentSize = 0

	// Start background compression
	go l.compressAndCleanup(rotatedPath)

	return nil
}

// reopenOrDie tries to reopen the log file after a failed rotation.
// If it can't, sets l.file to nil (writes will go to stderr).
func (l *RotatingLogger) reopenOrDie() {
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Cannot reopen log file %s: %v\n", l.path, err)
		l.file = nil
		return
	}
	l.file = file
	if info, err := l.file.Stat(); err == nil {
		l.currentSize = info.Size()
	}
}

// compressAndCleanup compresses a rotated log file and cleans up old logs
func (l *RotatingLogger) compressAndCleanup(rotatedPath string) {
	if err := compressLogFile(rotatedPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compress log %s: %v\n", rotatedPath, err)
	}
	if err := cleanupOldLogs(l.path, l.maxFiles); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to cleanup old logs: %v\n", err)
	}
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
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}

	gzPath := path + ".gz"
	dst, err := os.Create(gzPath)
	if err != nil {
		_ = src.Close()
		return fmt.Errorf("failed to create compressed file: %w", err)
	}

	gzWriter := gzip.NewWriter(dst)

	_, copyErr := io.Copy(gzWriter, src)
	closeGzErr := gzWriter.Close()
	_ = dst.Close()
	_ = src.Close()

	if copyErr != nil {
		_ = os.Remove(gzPath)
		return fmt.Errorf("failed to compress: %w", copyErr)
	}
	if closeGzErr != nil {
		_ = os.Remove(gzPath)
		return fmt.Errorf("failed to close gzip writer: %w", closeGzErr)
	}

	// Remove original - on Windows this can fail, retry once after a short delay
	if err := os.Remove(path); err != nil {
		time.Sleep(500 * time.Millisecond)
		if err := os.Remove(path); err != nil {
			// Not fatal - the .gz exists, uncompressed file is just wasted space
			fmt.Fprintf(os.Stderr, "Warning: could not remove %s after compression: %v\n", path, err)
		}
	}

	return nil
}

// cleanupOldLogs removes old rotated log files, keeping only maxFiles
func cleanupOldLogs(basePath string, maxFiles int) error {
	dir := filepath.Dir(basePath)
	baseName := filepath.Base(basePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	var rotatedLogs []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, baseName+".") && name != baseName {
			rotatedLogs = append(rotatedLogs, entry)
		}
	}

	if len(rotatedLogs) <= maxFiles {
		return nil
	}

	sort.Slice(rotatedLogs, func(i, j int) bool {
		return rotatedLogs[i].Name() < rotatedLogs[j].Name()
	})

	filesToRemove := len(rotatedLogs) - maxFiles
	for i := 0; i < filesToRemove; i++ {
		filePath := filepath.Join(dir, rotatedLogs[i].Name())
		if err := os.Remove(filePath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove old log %s: %v\n", filePath, err)
		}
	}

	return nil
}
