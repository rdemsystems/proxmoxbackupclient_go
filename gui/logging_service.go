//go:build service
// +build service

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var debugLogPath string

func init() {
	// Setup debug log path for SERVICE
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	logDir := filepath.Join(programData, "NimbusBackup")
	// #nosec G703 -- ProgramData is a trusted Windows system environment variable
	_ = os.MkdirAll(logDir, 0700)
	debugLogPath = filepath.Join(logDir, "debug-service.log")  // ← SERVICE LOG
}

func writeDebugLog(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[SERVICE %s] %s\n", timestamp, message)  // ← Prefix SERVICE

	f, err := os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write debug log: %v\n", err)
		return
	}
	defer f.Close()
	f.WriteString(logLine)

	fmt.Fprint(os.Stderr, logLine)
}
