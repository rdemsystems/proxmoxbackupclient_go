package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tizbac/proxmoxbackupclient_go/gui/api"
)

var debugLogPath string

func init() {
	// Setup debug log path
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	logDir := filepath.Join(programData, "NimbusBackup")
	// #nosec G703 -- ProgramData is a trusted Windows system environment variable, not user input
	_ = os.MkdirAll(logDir, 0700)
	debugLogPath = filepath.Join(logDir, "debug-gui.log")
}

func writeDebugLog(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)

	// Write to file
	f, err := os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write debug log: %v\n", err)
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.WriteString(logLine)

	// Also write to stderr for console visibility
	fmt.Fprint(os.Stderr, logLine)
}

// App struct contains the application state
type App struct {
	ctx              context.Context
	config           *Config
	stopScheduler    chan struct{}
	apiClient        *api.Client
	mode             api.ExecutionMode
	callbacksMap     map[string]*progressCallbacks
	callbacksMutex   sync.RWMutex
	isServiceProcess bool // True if running as Windows Service (never re-detect mode)
}

// progressCallbacks stores the callback functions for a backup operation
type progressCallbacks struct {
	onProgress func(jobID string, percent float64, message string)
	onComplete func(jobID string, success bool, message string)
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		config:        LoadConfig(),
		stopScheduler: make(chan struct{}),
		apiClient:     api.NewClient(),
		callbacksMap:  make(map[string]*progressCallbacks),
	}
}

// NewAppForService creates an App instance for Windows Service (no Wails runtime)
func NewAppForService(ctx context.Context) *App {
	return &App{
		ctx:              ctx,
		config:           LoadConfig(),
		stopScheduler:    make(chan struct{}),
		apiClient:        api.NewClient(),
		mode:             api.ModeStandalone, // Service executes directly
		callbacksMap:     make(map[string]*progressCallbacks),
		isServiceProcess: true, // Prevent mode re-detection
	}
}
