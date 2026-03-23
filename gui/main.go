//go:build !service
// +build !service

package main

import (
	"context"
	"embed"
	"flag"
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
	"security"

	"github.com/tizbac/proxmoxbackupclient_go/gui/api"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	appName = "Nimbus Backup"
)

// Version injected at build time via ldflags (-X main.appVersion=x.y.z)
// Source of truth: gui/wails.json productVersion
var appVersion = "dev" // Default for local dev without ldflags

var (
	debugLogPath    string
	crashReportPath string
)

func init() {

	// Get executable directory for crash reports
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	crashReportPath = filepath.Join(exeDir, "crash_report.txt")

	// Setup debug log file in ProgramData (Windows) or user home (others)
	// ProgramData is accessible by both GUI (user) and Service (LocalSystem)
	var logDir string
	if programData := os.Getenv("ProgramData"); programData != "" {
		// Windows: use C:\ProgramData\NimbusBackup (shared between GUI and Service)
		logDir = filepath.Join(programData, "NimbusBackup")
	} else if systemDrive := os.Getenv("SystemDrive"); systemDrive != "" {
		// Windows fallback: if ProgramData not set, use C:\ProgramData hardcoded
		// This ensures service logs are accessible even if env var is missing
		logDir = filepath.Join(systemDrive, "ProgramData", "NimbusBackup")
	} else {
		// Unix-like: use ~/.nimbus-backup
		homeDir, _ := os.UserHomeDir()
		logDir = filepath.Join(homeDir, ".nimbus-backup")
	}

	// Validate path for security (prevent path traversal)
	if err := security.ValidatePath(logDir); err != nil {
		// Fallback to current directory if path is invalid
		logDir = "."
	}

	// #nosec G703 -- Path is validated with security.ValidatePath() to prevent traversal
	// This is a legitimate use case: creating app log directory in user's home/appdata
	_ = os.MkdirAll(logDir, 0700)

	// Use different log files for GUI and Service to avoid concurrent write issues
	if IsServiceMode() {
		debugLogPath = filepath.Join(logDir, "debug-service.log")
	} else {
		debugLogPath = filepath.Join(logDir, "debug-gui.log")
	}

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
	// Parse command line flags
	minimized := flag.Bool("minimized", false, "Start minimized to system tray")
	flag.Parse()

	// Check for single instance (GUI only)
	// If another instance exists, activate it and exit
	if !CheckSingleInstance() {
		fmt.Println("Another instance is already running. Activating existing window...")
		os.Exit(0)
	}

	// Setup panic recovery for main
	defer func() {
		if r := recover(); r != nil {
			crashMsg := fmt.Sprintf("PANIC in main: %v\n%s", r, debug.Stack())
			writeDebugLog(crashMsg)
			writeCrashReport(crashMsg)
			fmt.Fprint(os.Stderr, "\n!!! APPLICATION CRASHED !!!\nSee crash_report.txt for details\n")
			os.Exit(1)
		}
	}()

	// Setup logging to both file and stderr
	logFile, err := os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
	} else {
		defer func() { _ = logFile.Close() }()
		// Log to both file and stderr
		log.SetOutput(io.MultiWriter(logFile, os.Stderr))
	}

	writeDebugLog(fmt.Sprintf("=== %s v%s Starting ===", appName, appVersion))
	writeDebugLog(fmt.Sprintf("Time: %s", time.Now().Format(time.RFC3339)))
	writeDebugLog(fmt.Sprintf("Debug log: %s", debugLogPath))
	writeDebugLog(fmt.Sprintf("Crash report path: %s", crashReportPath))

	// Clean up legacy auto-start from previous versions
	// (Task Scheduler or Registry entries before MSI service)
	CleanupLegacyAutoStart()

	// Create app instance
	app := NewApp()
	writeDebugLog("App instance created")

	// Create application options
	appOptions := &options.App{
		Title:     fmt.Sprintf("%s v%s", appName, appVersion),
		Width:     1200,
		Height:    800,
		MinWidth:  400,  // Allow very small windows for low-res screens
		MinHeight: 300,  // Allow very small windows for low-res screens
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		StartHidden:      *minimized, // Start hidden if --minimized flag is set
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

	if *minimized {
		writeDebugLog("Starting in minimized mode (hidden to tray)")
	}

	writeDebugLog("Application options configured")

	// Run application
	writeDebugLog("Starting Wails runtime...")
	err = wails.Run(appOptions)

	if err != nil {
		errMsg := fmt.Sprintf("ERROR: Wails.Run failed: %v\nStack trace:\n%s", err, debug.Stack())
		writeDebugLog(errMsg)
		writeCrashReport(errMsg)
		fmt.Fprint(os.Stderr, "\n!!! APPLICATION FAILED TO START !!!\n")
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
	defer func() { _ = f.Close() }()
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


// SetProgressCallbacks sets custom progress callbacks for API mode
func (a *App) SetProgressCallbacks(jobID string, onProgress func(string, float64, string), onComplete func(string, bool, string)) {
	writeDebugLog(fmt.Sprintf("[SetProgressCallbacks] Registered callbacks for jobID: %s", jobID))
	a.callbacksMutex.Lock()
	a.callbacksMap[jobID] = &progressCallbacks{
		onProgress: onProgress,
		onComplete: onComplete,
	}
	a.callbacksMutex.Unlock()
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	writeDebugLog("App.startup() called")

	// Detect execution mode (Service vs Standalone)
	detector := api.NewModeDetector()
	a.mode = detector.DetectMode()
	writeDebugLog(fmt.Sprintf("Execution mode: %s", a.mode.String()))

	// If running in standalone mode, start local scheduler
	// If in service mode, scheduler runs in the service
	if a.mode == api.ModeStandalone {
		// Cleanup any abandoned "running" jobs from previous session
		a.CleanupAbandonedJobs()

		// Start background job scheduler
		a.StartScheduler()
		writeDebugLog("Background scheduler started (standalone mode)")
	} else {
		writeDebugLog("Service mode detected - scheduler runs in service")
	}

	// Execute startup jobs (jobs with runAtStartup=true)
	// Note: In service mode, these will be sent via API
	go a.HandleStartupRun()

	// Setup system tray for background operation
	go a.SetupSystemTray()
}

// domReady is called after front-end resources have been loaded
func (a *App) domReady(ctx context.Context) {
	writeDebugLog("App.domReady() called - UI loaded successfully")
}

// beforeClose is called when the application is about to quit
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	writeDebugLog("App.beforeClose() called - minimizing to tray")
	// Instead of closing, minimize to tray
	a.MinimizeToTray()
	return true // Prevent actual close
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

// GetSystemInfo returns system information for UI (mode, admin status, etc.)
func (a *App) GetSystemInfo() map[string]interface{} {
	return map[string]interface{}{
		"mode":     a.mode.String(),
		"is_admin": isAdmin(),
		"hostname": a.GetHostname(),
		"service_available": a.mode == api.ModeService,
	}
}

func (a *App) GetVersion() string {
	writeDebugLog(fmt.Sprintf("GetVersion() returned: %s", appVersion))
	return appVersion
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

// DiagnoseConfig returns config validation status for debugging
func (a *App) DiagnoseConfig() map[string]interface{} {
	cfg := a.GetConfig()

	var validationError string
	if err := cfg.Validate(); err != nil {
		validationError = err.Error()
	}

	configPath, _ := getConfigPath()

	return map[string]interface{}{
		"config_path":       configPath,
		"baseurl_set":       cfg.BaseURL != "",
		"baseurl_value":     security.SanitizeURL(cfg.BaseURL),
		"authid_set":        cfg.AuthID != "",
		"datastore_set":     cfg.Datastore != "",
		"validation_ok":     validationError == "",
		"validation_error":  validationError,
		"mode":              a.mode.String(),
	}
}

// SaveConfig saves the configuration
func (a *App) SaveConfig(config *Config) error {
	// Log sanitized config (no secrets)
	writeDebugLog(fmt.Sprintf("SaveConfig() called: URL=%s, AuthID=%s, Datastore=%s, BackupID=%s",
		security.SanitizeURL(config.BaseURL),
		config.AuthID,
		config.Datastore,
		config.BackupID))

	// Validate before saving
	if err := config.Validate(); err != nil {
		writeDebugLog(fmt.Sprintf("Config validation failed: %v", err))
		return err
	}

	// Save to disk
	if err := config.Save(); err != nil {
		writeDebugLog(fmt.Sprintf("Config save to disk failed: %v", err))
		return err
	}

	// Update in-memory config
	a.config = config
	writeDebugLog("Config saved successfully and loaded into app")
	return nil
}

// TestConnection tests the PBS connection with the provided config (or current if nil)
func (a *App) TestConnection(config *Config) error {
	writeDebugLog("TestConnection() called")

	// Use provided config or fallback to current app config
	testConfig := config
	if testConfig == nil {
		testConfig = a.config
	}

	// Validate config first
	if err := testConfig.Validate(); err != nil {
		return err
	}

	// Create PBS client
	client := &pbscommon.PBSClient{
		BaseURL:         testConfig.BaseURL,
		CertFingerPrint: testConfig.CertFingerprint,
		AuthID:          testConfig.AuthID,
		Secret:          testConfig.Secret,
		Datastore:       testConfig.Datastore,
		Namespace:       testConfig.Namespace,
		Insecure:        testConfig.CertFingerprint != "",
		Manifest: pbscommon.BackupManifest{
			BackupID: testConfig.BackupID,
		},
	}

	// Debug log with sanitized credentials
	writeDebugLog(fmt.Sprintf("Testing connection: URL=%s, AuthID=%s, Secret=%s, Datastore=%s",
		security.SanitizeURL(testConfig.BaseURL),
		testConfig.AuthID,
		security.SanitizeSecret(testConfig.Secret),
		testConfig.Datastore))

	// Perform real HTTP test (checks DNS, connectivity, auth, datastore access)
	if err := client.TestConnection(); err != nil {
		writeDebugLog(fmt.Sprintf("Connection test failed: %v", err))
		return err
	}

	writeDebugLog("Connection test successful (authenticated + datastore accessible)")
	return nil
}

// GetLastBackupDirs returns the last used backup directories
func (a *App) GetLastBackupDirs() []string {
	writeDebugLog(fmt.Sprintf("GetLastBackupDirs() returned %d directories", len(a.config.LastBackupDirs)))
	return a.config.LastBackupDirs
}

// ReloadConfig reloads configuration from disk (for service when config changes)
func (a *App) ReloadConfig() {
	newConfig := LoadConfig()
	a.config = newConfig
	writeDebugLog("Config reloaded from disk")
}

// ==================== MULTI-PBS MANAGEMENT ====================

// ListPBSServers returns all configured PBS servers
func (a *App) ListPBSServers() []*PBSServer {
	writeDebugLog(fmt.Sprintf("ListPBSServers() returned %d servers", len(a.config.PBSServers)))
	return a.config.ListPBSServers()
}

// GetPBSServer returns a single PBS server by ID
func (a *App) GetPBSServer(id string) (*PBSServer, error) {
	writeDebugLog(fmt.Sprintf("GetPBSServer(%s) called", id))
	return a.config.GetPBSServer(id)
}

// AddPBSServer adds a new PBS server to the configuration
func (a *App) AddPBSServer(pbs *PBSServer) error {
	writeDebugLog(fmt.Sprintf("AddPBSServer(%s) called", pbs.ID))
	return a.config.AddPBSServer(pbs)
}

// UpdatePBSServer updates an existing PBS server
func (a *App) UpdatePBSServer(pbs *PBSServer) error {
	writeDebugLog(fmt.Sprintf("UpdatePBSServer(%s) called", pbs.ID))
	return a.config.UpdatePBSServer(pbs)
}

// DeletePBSServer removes a PBS server
func (a *App) DeletePBSServer(id string) error {
	writeDebugLog(fmt.Sprintf("DeletePBSServer(%s) called", id))
	return a.config.DeletePBSServer(id)
}

// SetDefaultPBSServer sets the default PBS server
func (a *App) SetDefaultPBSServer(id string) error {
	writeDebugLog(fmt.Sprintf("SetDefaultPBSServer(%s) called", id))
	return a.config.SetDefaultPBS(id)
}

// GetDefaultPBSID returns the default PBS server ID
func (a *App) GetDefaultPBSID() string {
	return a.config.DefaultPBSID
}

// TestPBSConnection tests connection to a specific PBS server
func (a *App) TestPBSConnection(pbsID string) error {
	writeDebugLog(fmt.Sprintf("TestPBSConnection(%s) called", pbsID))

	pbs, err := a.config.GetPBSServer(pbsID)
	if err != nil {
		return err
	}

	// Convert to legacy Config format for existing TestConnection logic
	legacyConfig := pbs.ToConfig()
	return a.TestConnection(legacyConfig)
}

// ==================== END MULTI-PBS MANAGEMENT ====================

// StartBackup starts a backup operation (routes to service or direct based on mode)
func (a *App) StartBackup(backupType string, backupDirs []string, driveLetters []string, excludeList []string, backupID string, useVSS bool) error {
	writeDebugLog(fmt.Sprintf("StartBackup() called - mode: %s, VSS: %v, isServiceProcess: %v", a.mode.String(), useVSS, a.isServiceProcess))

	// Re-detect mode if currently Standalone (service may have started after GUI)
	// IMPORTANT: Never re-detect if we ARE the service process (prevents infinite loop)
	if !a.isServiceProcess && a.mode == api.ModeStandalone {
		if a.apiClient.IsServiceAvailable() {
			writeDebugLog("[Mode Detection] Service now available, switching to Service mode")
			a.mode = api.ModeService
		}
	}

	// Route based on execution mode
	switch a.mode {
	case api.ModeService:
		// Use HTTP API to communicate with service (service has admin rights as LocalSystem)
		return a.startBackupViaService(backupType, backupDirs, driveLetters, excludeList, backupID, useVSS)
	case api.ModeStandalone:
		// Direct execution - check admin if VSS requested
		if useVSS && !isAdmin() {
			return fmt.Errorf("VSS (Shadow Copy) nécessite les privilèges administrateur - veuillez redémarrer l'application en tant qu'administrateur ou désactiver VSS")
		}
		return a.startBackupDirect(backupType, backupDirs, driveLetters, excludeList, backupID, useVSS)
	default:
		return fmt.Errorf("unknown execution mode: %v", a.mode)
	}
}

// startBackupViaService sends backup request to the service via HTTP API
func (a *App) startBackupViaService(backupType string, backupDirs []string, driveLetters []string, excludeList []string, backupID string, useVSS bool) error {
	writeDebugLog("[Service Mode] Sending backup request to service")

	req := &api.BackupRequest{
		BackupType:   backupType,
		BackupID:     backupID,
		BackupDirs:   backupDirs,
		DriveLetters: driveLetters,
		ExcludeList:  excludeList,
		UseVSS:       useVSS,
	}

	resp, err := a.apiClient.StartBackup(req)
	if err != nil {
		writeDebugLog(fmt.Sprintf("[Service Mode] Backup request failed: %v", err))
		return fmt.Errorf("échec de la communication avec le service: %w", err)
	}

	writeDebugLog(fmt.Sprintf("[Service Mode] Backup started: %s (JobID: %s)", resp.Message, resp.JobID))

	// Start polling for progress updates
	go a.pollBackupProgress(resp.JobID)

	return nil
}

// pollBackupProgress polls the service for backup progress and emits events to GUI
func (a *App) pollBackupProgress(jobID string) {
	writeDebugLog(fmt.Sprintf("[Service Mode] Starting progress polling for job: %s", jobID))
	ticker := time.NewTicker(3 * time.Second) // Poll every 3 seconds
	defer ticker.Stop()

	for range ticker.C {
		progress, err := a.apiClient.GetBackupStatus(jobID)
		if err != nil {
			writeDebugLog(fmt.Sprintf("[Service Mode] Failed to get progress: %v", err))
			continue
		}

		// Emit progress event to GUI
		if a.ctx != nil && progress.Running {
			runtime.EventsEmit(a.ctx, "backup:progress", map[string]interface{}{
				"percent": progress.Progress,
				"message": progress.Message,
			})
		}

		// If backup completed, emit final event and stop polling
		if progress.Complete {
			writeDebugLog(fmt.Sprintf("[Service Mode] Backup completed: success=%v", progress.Success))
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "backup:complete", map[string]interface{}{
					"success": progress.Success,
					"message": progress.Message,
				})
			}
			return
		}
	}
}

// startBackupDirect performs backup directly (standalone mode)
func (a *App) startBackupDirect(backupType string, backupDirs []string, driveLetters []string, excludeList []string, backupID string, useVSS bool) error {
	// Use hostname as fallback if backupID is empty
	if backupID == "" {
		backupID = a.GetHostname()
		writeDebugLog(fmt.Sprintf("[Backup ID] Empty backup-id, using hostname: %s", backupID))
	}

	// Sanitize backup ID for logging
	sanitizedID := security.SanitizeForLog(backupID)
	writeDebugLog(fmt.Sprintf("[Standalone Mode] StartBackup: type=%s, id=%s, vss=%v, dir_count=%d",
		backupType, sanitizedID, useVSS, len(backupDirs)))

	// Validate BackupID (now guaranteed to be non-empty)
	if err := security.ValidateBackupID(backupID); err != nil {
		return fmt.Errorf("backup ID invalide: %w", err)
	}

	// Validate backup directories
	for _, dir := range backupDirs {
		if err := security.ValidatePath(dir); err != nil {
			return fmt.Errorf("chemin invalide '%s': %w", dir, err)
		}
	}

	// Note: Admin check for VSS is done in StartBackup() routing layer
	// If we're here via service, we're already running as LocalSystem

	// Validate PBS config
	if err := a.config.Validate(); err != nil {
		return err
	}

	// Validate backup parameters and build target list
	var targetDirs []string
	if backupType == "directory" {
		if len(backupDirs) == 0 {
			return fmt.Errorf("au moins un répertoire de sauvegarde requis")
		}
		targetDirs = backupDirs
	}
	if backupType == "machine" {
		if len(driveLetters) == 0 {
			return fmt.Errorf("au moins un disque physique requis")
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

			// Check if there's a registered callback for any job (service mode)
			a.callbacksMutex.RLock()
			hasCallbacks := len(a.callbacksMap) > 0
			if hasCallbacks {
				// Call all registered callbacks (typically just one per backup)
				for jobID, callbacks := range a.callbacksMap {
					if callbacks.onProgress != nil {
						writeDebugLog(fmt.Sprintf("[OnProgress] Calling custom callback for jobID: %s", jobID))
						callbacks.onProgress(jobID, percent*100, message)
					}
				}
			}
			a.callbacksMutex.RUnlock()

			// If no custom callbacks and we have Wails context, emit events (GUI standalone mode)
			// NEVER emit events if we're the service process (no Wails runtime)
			if !hasCallbacks && !a.isServiceProcess && a.ctx != nil {
				writeDebugLog("[OnProgress] Emitting Wails event (GUI mode)")
				runtime.EventsEmit(a.ctx, "backup:progress", map[string]interface{}{
					"percent": percent * 100,
					"message": message,
				})
			} else if !hasCallbacks && (a.isServiceProcess || a.ctx == nil) {
				writeDebugLog("[OnProgress] No callbacks/context (service or headless mode)")
			}
		},
		OnComplete: func(success bool, message string) {
			writeDebugLog(fmt.Sprintf("Backup complete: success=%v, %s", success, message))

			// Check if there's a registered callback for any job (service mode)
			a.callbacksMutex.RLock()
			hasCallbacks := len(a.callbacksMap) > 0
			var jobIDsToCleanup []string
			if hasCallbacks {
				// Call all registered callbacks and collect jobIDs for cleanup
				for jobID, callbacks := range a.callbacksMap {
					if callbacks.onComplete != nil {
						writeDebugLog(fmt.Sprintf("[OnComplete] Calling custom callback for jobID: %s", jobID))
						callbacks.onComplete(jobID, success, message)
					}
					jobIDsToCleanup = append(jobIDsToCleanup, jobID)
				}
			}
			a.callbacksMutex.RUnlock()

			// Clean up completed callbacks
			if len(jobIDsToCleanup) > 0 {
				a.callbacksMutex.Lock()
				for _, jobID := range jobIDsToCleanup {
					delete(a.callbacksMap, jobID)
					writeDebugLog(fmt.Sprintf("[OnComplete] Cleaned up callbacks for jobID: %s", jobID))
				}
				a.callbacksMutex.Unlock()
			}

			// If no custom callbacks and we have Wails context, emit events (GUI standalone mode)
			// NEVER emit events if we're the service process (no Wails runtime)
			if !hasCallbacks && !a.isServiceProcess && a.ctx != nil {
				writeDebugLog("[OnComplete] Emitting Wails event (GUI mode)")
				runtime.EventsEmit(a.ctx, "backup:complete", map[string]interface{}{
					"success": success,
					"message": message,
				})
			} else if !hasCallbacks && (a.isServiceProcess || a.ctx == nil) {
				writeDebugLog("[OnComplete] No callbacks/context (service or headless mode)")
			}

			// Add manual backup to history
			historyEntry := JobHistory{
				ID:         fmt.Sprintf("%d", time.Now().Unix()),
				Name:       fmt.Sprintf("Backup manuel - %s", backupID),
				Timestamp:  time.Now().Format(time.RFC3339),
				Status:     "success",
				Message:    message,
				BackupDirs: targetDirs,
				BackupID:   backupID,
				UseVSS:     useVSS,
			}
			if !success {
				historyEntry.Status = "failed"
			}
			if err := a.AddJobHistory(historyEntry); err != nil {
				writeDebugLog(fmt.Sprintf("Warning: Failed to add manual backup to history: %v", err))
			}

			// Save last used backup directories on success
			if success && backupType == "directory" {
				a.config.LastBackupDirs = backupDirs
				if err := a.config.Save(); err != nil {
					writeDebugLog(fmt.Sprintf("Failed to save last backup dirs: %v", err))
				} else {
					writeDebugLog(fmt.Sprintf("Saved %d backup directories to config", len(backupDirs)))
				}
			}
		},
	}

	// Run backup inline (in background goroutine to not block UI)
	go func() {
		// Machine backup disabled for now - Windows Defender flags it
		// if backupType == "machine" {
		// 	err = RunMachineBackup(opts)
		// } else {
		err := RunBackupInline(opts)
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
		return nil, fmt.Errorf("échec de la liste des snapshots: %v", err)
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
		return fmt.Errorf("chemin de destination requis")
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
		return fmt.Errorf("échec de la restauration: %v", err)
	}

	writeDebugLog("Restore completed successfully")
	return nil
}
