package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/tizbac/proxmoxbackupclient_go/pkg/logger"
)

func TestLogger_Init(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Init(buf, slog.LevelDebug)

	log := logger.Get()
	if log == nil {
		t.Fatal("logger should not be nil after Init")
	}
}

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Level
		logFunc  func(string, ...any)
		expected bool // should appear in output
	}{
		{"debug at debug level", slog.LevelDebug, logger.Debug, true},
		{"info at debug level", slog.LevelDebug, logger.Info, true},
		{"warn at debug level", slog.LevelDebug, logger.Warn, true},
		{"error at debug level", slog.LevelDebug, logger.Error, true},
		{"debug at info level", slog.LevelInfo, logger.Debug, false},
		{"info at info level", slog.LevelInfo, logger.Info, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger.Init(buf, tt.level)

			tt.logFunc("test message", "key", "value")

			output := buf.String()
			hasOutput := strings.Contains(output, "test message")

			if hasOutput != tt.expected {
				t.Errorf("expected output=%v, got=%v (output: %s)", tt.expected, hasOutput, output)
			}
		})
	}
}

func TestLogger_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Init(buf, slog.LevelInfo)

	logger.Info("test message", "key", "value", "number", 42)

	output := buf.String()

	// Should be valid JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Check required fields
	if logEntry["msg"] != "test message" {
		t.Errorf("expected msg='test message', got=%v", logEntry["msg"])
	}

	if logEntry["key"] != "value" {
		t.Errorf("expected key='value', got=%v", logEntry["key"])
	}

	if logEntry["number"] != float64(42) {
		t.Errorf("expected number=42, got=%v", logEntry["number"])
	}
}

func TestLogger_With(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Init(buf, slog.LevelInfo)

	baseLogger := logger.Get()
	contextLogger := baseLogger.With("request_id", "abc123")

	contextLogger.Info("test message")

	output := buf.String()

	if !strings.Contains(output, "request_id") {
		t.Error("output should contain request_id")
	}

	if !strings.Contains(output, "abc123") {
		t.Error("output should contain abc123")
	}
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Init(buf, slog.LevelInfo)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info("concurrent message", "id", id)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have some output
	if buf.Len() == 0 {
		t.Error("expected some log output from concurrent access")
	}
}
