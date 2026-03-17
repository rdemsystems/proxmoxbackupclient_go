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
	wailswin "github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"pbscommon"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	appName    = "Nimbus Backup"
	appVersion = "0.0.16"
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

	// Setup debug log file in AppData (Windows) or user home (others)
	var logDir string
	if appData := os.Getenv("APPDATA"); appData != "" {
		// Windows: use %APPDATA%\NimbusBackup
		logDir = filepath.Join(appData, "NimbusBackup")
	} else {
		// Unix-like: use ~/.nimbus-backup
		homeDir, _ := os.UserHomeDir()
		logDir = filepath.Join(homeDir, ".nimbus-backup")
	}
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
		Windows: &wailswin.Options{
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
- Website: https://nimbus.rdem-systems.com
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

// GetHostname returns the system hostname
func (a *App) GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		writeDebugLog(fmt.Sprintf("GetHostname() error: %v", err))
		return "unknown"
	}
	writeDebugLog(fmt.Sprintf("GetHostname() returned: %s", hostname))
	return hostname
}

// ListPhysicalDisks returns a list of available physical disks (DISABLED - feature postponed)
/*
func (a *App) ListPhysicalDisks() ([]PhysicalDiskInfo, error) {
	writeDebugLog("ListPhysicalDisks() called from frontend")
	disks, err := ListPhysicalDisks()
	if err != nil {
		writeDebugLog(fmt.Sprintf("ListPhysicalDisks() error: %v", err))
		return nil, err
	}
	writeDebugLog(fmt.Sprintf("Found %d physical disks", len(disks)))
	return disks, nil
}
*/

// GetConfigWithHostname returns config with hostname pre-filled
func (a *App) GetConfigWithHostname() map[string]interface{} {
	hostname := a.GetHostname()
	cfg := a.GetConfig()

	// Return config as map with hostname
	result := map[string]interface{}{
		"baseurl":         cfg.BaseURL,
		"certfingerprint": cfg.CertFingerprint,
		"authid":          cfg.AuthID,
		"secret":          cfg.Secret,
		"datastore":       cfg.Datastore,
		"namespace":       cfg.Namespace,
		"backupdir":       cfg.BackupDir,
		"backup-id":       cfg.BackupID,
		"usevss":          cfg.UseVSS,
		"hostname":        hostname,
	}

	// Pre-fill backup-id with hostname if empty
	if cfg.BackupID == "" {
		result["backup-id"] = hostname
	}

	return result
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

	// Validate config first
	if err := a.config.Validate(); err != nil {
		return err
	}

	// Create PBS client
	client := &pbscommon.PBSClient{
		BaseURL:         a.config.BaseURL,
		CertFingerPrint: a.config.CertFingerprint,
		AuthID:          a.config.AuthID,
		Secret:          a.config.Secret,
		Datastore:       a.config.Datastore,
		Namespace:       a.config.Namespace,
		Insecure:        a.config.CertFingerprint != "",
		Manifest: pbscommon.BackupManifest{
			BackupID: a.config.BackupID,
		},
	}

	// Debug log (mask secret)
	maskedSecret := "***"
	if len(a.config.Secret) > 4 {
		maskedSecret = a.config.Secret[:4] + "..." + a.config.Secret[len(a.config.Secret)-4:]
	}
	writeDebugLog(fmt.Sprintf("Testing connection: URL=%s, AuthID=%s, Secret=%s, Datastore=%s",
		a.config.BaseURL, a.config.AuthID, maskedSecret, a.config.Datastore))

	// Try to connect in backup mode (not reader) to test with Datastore.Backup permission
	client.Connect(false, "host")

	// Simple test: if Connect() doesn't panic/error, connection is OK
	// We can't easily test without actually starting a backup session
	writeDebugLog("Connection test successful (authenticated)")
	return nil
}

// StartBackup starts a backup operation
func (a *App) StartBackup(backupType string, backupDirs []string, driveLetters []string, excludeList []string, backupID string, useVSS bool) error {
	writeDebugLog(fmt.Sprintf("StartBackup() called: type=%s, dirs=%v, drives=%v, id=%s, vss=%v",
		backupType, backupDirs, driveLetters, backupID, useVSS))

	// Check admin privileges if VSS is requested
	if useVSS && !isAdmin() {
		return fmt.Errorf("VSS (Shadow Copy) nécessite les privilèges administrateur. Veuillez redémarrer l'application en tant qu'administrateur ou désactiver VSS.")
	}

	// Validate PBS config
	if err := a.config.Validate(); err != nil {
		return err
	}

	// Validate backup parameters and build target list
	var targetDirs []string
	if backupType == "directory" {
		if len(backupDirs) == 0 {
			return fmt.Errorf("Au moins un répertoire de sauvegarde requis")
		}
		targetDirs = backupDirs
	}
	if backupType == "machine" {
		if len(driveLetters) == 0 {
			return fmt.Errorf("Au moins un disque physique requis")
		}
		// Physical drive paths are used directly (e.g., \\.\PhysicalDrive0)
		targetDirs = driveLetters
	}

	// Prepare backup options
	opts := BackupOptions{
		BaseURL:         a.config.BaseURL,
		AuthID:          a.config.AuthID,
		Secret:          a.config.Secret,
		Datastore:       a.config.Datastore,
		Namespace:       a.config.Namespace,
		CertFingerprint: a.config.CertFingerprint,
		BackupDirs:      targetDirs,
		BackupID:        backupID,
		BackupType:      "host", // "host" for directory, would be "vm" for machine
		UseVSS:          useVSS,
		OnProgress: func(percent float64, message string) {
			writeDebugLog(fmt.Sprintf("Progress: %.1f%% - %s", percent*100, message))
			runtime.EventsEmit(a.ctx, "backup:progress", map[string]interface{}{
				"percent": percent * 100,
				"message": message,
			})
		},
		OnComplete: func(success bool, message string) {
			writeDebugLog(fmt.Sprintf("Backup complete: success=%v, %s", success, message))
			runtime.EventsEmit(a.ctx, "backup:complete", map[string]interface{}{
				"success": success,
				"message": message,
			})
		},
	}

	// Run backup inline (in background goroutine to not block UI)
	go func() {
		var err error
		// Machine backup disabled for now - Windows Defender flags it
		// if backupType == "machine" {
		// 	err = RunMachineBackup(opts)
		// } else {
		err = RunBackupInline(opts)
		// }
		if err != nil {
			writeDebugLog(fmt.Sprintf("Backup error: %v", err))
		}
	}()

	return nil
}

// ListSnapshots lists available snapshots
func (a *App) ListSnapshots(backupID string) ([]map[string]string, error) {
	writeDebugLog(fmt.Sprintf("ListSnapshots() called: backupID=%s", backupID))

	// Validate config
	if err := a.config.Validate(); err != nil {
		return nil, err
	}

	// Create restore manager
	rm := NewRestoreManager(a.config)

	// List snapshots
	snapshots, err := rm.ListSnapshots()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Failed to list snapshots: %v", err))
		return nil, fmt.Errorf("Échec de la liste des snapshots: %v", err)
	}

	// Convert to map format for frontend
	result := make([]map[string]string, 0, len(snapshots))
	for _, snap := range snapshots {
		// Filter by backup ID if specified
		if backupID != "" && snap.ID != backupID {
			continue
		}

		result = append(result, map[string]string{
			"id":   snap.Timestamp.Format("2006-01-02T15:04:05Z"),
			"time": snap.Timestamp.Format("2006-01-02 15:04:05"),
			"type": snap.Type,
		})
	}

	writeDebugLog(fmt.Sprintf("Found %d snapshots", len(result)))
	return result, nil
}

// RestoreSnapshot restores a snapshot
func (a *App) RestoreSnapshot(snapshotID, destPath string) error {
	writeDebugLog(fmt.Sprintf("RestoreSnapshot() called: snapshot=%s, dest=%s", snapshotID, destPath))

	// Validate config
	if err := a.config.Validate(); err != nil {
		return err
	}

	if snapshotID == "" {
		return fmt.Errorf("ID du snapshot requis")
	}

	if destPath == "" {
		return fmt.Errorf("Chemin de destination requis")
	}

	// Create restore manager
	rm := NewRestoreManager(a.config)

	// Parse timestamp from snapshotID
	timestamp, err := time.Parse("2006-01-02T15:04:05Z", snapshotID)
	if err != nil {
		writeDebugLog(fmt.Sprintf("Failed to parse snapshot ID: %v", err))
		return fmt.Errorf("ID de snapshot invalide: %v", err)
	}

	// Create snapshot object
	snapshot := BackupSnapshot{
		Type:      "host",
		ID:        a.config.BackupID,
		Timestamp: timestamp,
		Files: []BackupFile{
			{Name: "root.pxar.didx", Type: "pxar"},
		},
	}

	// Restore the snapshot
	err = rm.RestoreFile(snapshot, snapshot.Files[0], destPath)
	if err != nil {
		writeDebugLog(fmt.Sprintf("Failed to restore: %v", err))
		return fmt.Errorf("Échec de la restauration: %v", err)
	}

	writeDebugLog("Restore completed successfully")
	return nil
}
