package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	appName    = "Nimbus Backup"
	appVersion = "0.4.0"
)

var (
	debugLogPath    string
	crashReportPath string
)

func init() {
	// Get executable directory for crash reports
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	crashReportPath = filepath.Join(exeDir, "crash_report.txt")

	// Setup debug log file in user home
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".proxmox-backup-guardian")
	_ = os.MkdirAll(logDir, 0700)
	debugLogPath = filepath.Join(logDir, "debug.log")

	// Setup panic recovery
	defer func() {
		if r := recover(); r != nil {
			crashMsg := fmt.Sprintf("PANIC during init: %v\n%s", r, debug.Stack())
			writeDebugLog(crashMsg)
			writeCrashReport(crashMsg)
		}
	}()
}

func main() {
	// Setup panic recovery for main
	defer func() {
		if r := recover(); r != nil {
			crashMsg := fmt.Sprintf("PANIC in main: %v\n%s", r, debug.Stack())
			writeDebugLog(crashMsg)
			writeCrashReport(crashMsg)
			fmt.Fprintf(os.Stderr, "\n!!! APPLICATION CRASHED !!!\nSee crash_report.txt for details\n")
			os.Exit(1)
		}
	}()

	// Setup logging to both file and stderr
	logFile, err := os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
	} else {
		defer logFile.Close()
		// Log to both file and stderr
		log.SetOutput(io.MultiWriter(logFile, os.Stderr))
	}

	writeDebugLog(fmt.Sprintf("=== %s v%s Starting ===", appName, appVersion))
	writeDebugLog(fmt.Sprintf("Time: %s", time.Now().Format(time.RFC3339)))
	writeDebugLog(fmt.Sprintf("Debug log: %s", debugLogPath))
	writeDebugLog(fmt.Sprintf("Crash report path: %s", crashReportPath))

	// Create app instance
	app := NewApp()
	writeDebugLog("App instance created")

	// Create application options
	appOptions := &options.App{
		Title:  fmt.Sprintf("%s v%s", appName, appVersion),
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			WebviewUserDataPath:  filepath.Join(os.Getenv("APPDATA"), "NimbusBackup"),
		},
	}

	writeDebugLog("Application options configured")

	// Run application
	writeDebugLog("Starting Wails runtime...")
	err = wails.Run(appOptions)

	if err != nil {
		errMsg := fmt.Sprintf("ERROR: Wails.Run failed: %v\nStack trace:\n%s", err, debug.Stack())
		writeDebugLog(errMsg)
		writeCrashReport(errMsg)
		fmt.Fprintf(os.Stderr, "\n!!! APPLICATION FAILED TO START !!!\n")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Check crash_report.txt and %s\n", debugLogPath)
		os.Exit(1)
	}

	writeDebugLog("Application shutdown normally")
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
	defer f.Close()
	_, _ = f.WriteString(logLine)

	// Also write to stderr for console visibility
	fmt.Fprint(os.Stderr, logLine)
}

func writeCrashReport(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	crashContent := fmt.Sprintf(`=== NIMBUS BACKUP CRASH REPORT ===
Time: %s
Version: %s

%s

=== SYSTEM INFO ===
Debug Log: %s

Please report this issue to RDEM Systems:
- Website: https://backup.rdem-systems.com
- Include this crash_report.txt file
`, timestamp, appVersion, message, debugLogPath)

	// Write to crash report file (overwrite each time)
	err := os.WriteFile(crashReportPath, []byte(crashContent), 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write crash report: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Crash report written to: %s\n", crashReportPath)
	}
}

// App struct
type App struct {
	ctx    context.Context
	config *Config
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		config: LoadConfig(),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	writeDebugLog("App.startup() called")
}

// domReady is called after front-end resources have been loaded
func (a *App) domReady(ctx context.Context) {
	writeDebugLog("App.domReady() called - UI loaded successfully")
}

// beforeClose is called when the application is about to quit
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	writeDebugLog("App.beforeClose() called")
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	writeDebugLog("App.shutdown() called")
}

// GetConfig returns the current configuration
func (a *App) GetConfig() *Config {
	writeDebugLog("GetConfig() called from frontend")
	return a.config
}

// SaveConfig saves the configuration
func (a *App) SaveConfig(config *Config) error {
	writeDebugLog(fmt.Sprintf("SaveConfig() called: %+v", config))
	a.config = config
	return config.Save()
}

// TestConnection tests the PBS connection
func (a *App) TestConnection() error {
	writeDebugLog("TestConnection() called")
	// TODO: Implement actual PBS connection test
	if a.config.BaseURL == "" {
		return fmt.Errorf("URL du serveur PBS requis")
	}
	return nil
}

// StartBackup starts a backup operation
func (a *App) StartBackup(backupType, backupDir, driveLetter string, excludeList []string, backupID string, useVSS bool) error {
	writeDebugLog(fmt.Sprintf("StartBackup() called: type=%s, dir=%s, drive=%s, id=%s, vss=%v",
		backupType, backupDir, driveLetter, backupID, useVSS))

	// TODO: Implement actual backup logic
	if err := a.config.Validate(); err != nil {
		return err
	}

	return fmt.Errorf("Fonctionnalité de backup à implémenter")
}

// ListSnapshots lists available snapshots
func (a *App) ListSnapshots(backupID string) ([]map[string]string, error) {
	writeDebugLog(fmt.Sprintf("ListSnapshots() called: backupID=%s", backupID))

	// TODO: Implement actual snapshot listing
	// Mock response for now
	return []map[string]string{
		{
			"id":   "2024-03-17T10:30:00Z",
			"time": "2024-03-17 10:30:00",
			"type": "machine",
		},
	}, nil
}

// RestoreSnapshot restores a snapshot
func (a *App) RestoreSnapshot(snapshotID, destPath string) error {
	writeDebugLog(fmt.Sprintf("RestoreSnapshot() called: snapshot=%s, dest=%s", snapshotID, destPath))

	// TODO: Implement actual restore logic
	return fmt.Errorf("Fonctionnalité de restore à implémenter")
}
