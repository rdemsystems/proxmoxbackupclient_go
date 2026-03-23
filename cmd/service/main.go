//go:build windows

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kardianos/service"

	// Import from gui package for App logic and API
	gui "github.com/tizbac/proxmoxbackupclient_go/gui"
	"github.com/tizbac/proxmoxbackupclient_go/gui/api"
)

const (
	serviceName        = "NimbusBackup"
	serviceDisplayName = "Nimbus Backup SVC"
	serviceDescription = "Executes scheduled backups to Proxmox Backup Server with VSS support"
	apiPort            = "18765" // Fixed port for GUI<->Service communication
)

var (
	logger      service.Logger
	debugLogPath string
)

// NimbusService wraps the application for Windows Service execution
type NimbusService struct {
	app       *gui.App
	apiServer *api.Server
	stopChan  chan struct{}
}

func init() {
	// Setup debug log file in ProgramData
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	logDir := filepath.Join(programData, "NimbusBackup")
	_ = os.MkdirAll(logDir, 0700)
	debugLogPath = filepath.Join(logDir, "debug-service.log")
}

// Start is called when the service starts
func (s *NimbusService) Start(svc service.Service) error {
	writeLog("Nimbus Backup service starting...")
	s.stopChan = make(chan struct{})
	go s.run()
	return nil
}

// run contains the main service loop
func (s *NimbusService) run() {
	writeLog("Nimbus Backup service running")

	// Initialize app with background context (service has no Wails runtime)
	s.app = gui.NewAppForService(context.Background())

	hostname, _ := os.Hostname()
	writeLog(fmt.Sprintf("Service running for host: %s", hostname))

	// Clean up orphaned VSS snapshots from previous crashes
	writeLog("Running VSS cleanup...")
	if err := cleanupVSS(); err != nil {
		writeLog(fmt.Sprintf("VSS cleanup failed (non-fatal): %v", err))
	}

	// Clean up abandoned jobs
	s.app.CleanupAbandonedJobs()

	// Start the scheduler
	s.app.StartScheduler()
	writeLog("Job scheduler started")

	// Start HTTP API server for GUI communication
	apiAddr := "127.0.0.1:" + apiPort
	s.apiServer = api.NewServer(apiAddr, s.app)
	writeLog(fmt.Sprintf("Starting HTTP API server on %s", apiAddr))

	go func() {
		if err := s.apiServer.Start(); err != nil {
			writeLog(fmt.Sprintf("API server error: %v", err))
		}
	}()

	// Keep the service running (scheduler and API server run in background goroutines)
	writeLog("Service main loop started, waiting for stop signal")
	<-s.stopChan
	writeLog("Stop signal received, service main loop exiting")
}

// Stop is called when the service stops
func (s *NimbusService) Stop(svc service.Service) error {
	writeLog("Nimbus Backup service stopping...")

	// Signal the main loop to exit
	if s.stopChan != nil {
		close(s.stopChan)
	}

	// Stop the scheduler gracefully
	if s.app != nil {
		s.app.StopScheduler()
	}

	// Give it a moment to finish current operations
	time.Sleep(2 * time.Second)

	writeLog("Nimbus Backup service stopped")
	return nil
}

func main() {
	// Command-line flags for service control
	svcFlag := flag.String("service", "", "Control the system service: install, uninstall, start, stop, restart")
	flag.Parse()

	// Service configuration
	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
	}

	nimbusSvc := &NimbusService{}
	s, err := service.New(nimbusSvc, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Setup logger
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}

	// Handle service control commands (install, uninstall, start, stop)
	if len(*svcFlag) != 0 {
		err := service.Control(s, *svcFlag)
		if err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
		writeLog(fmt.Sprintf("Service control action '%s' completed", *svcFlag))
		return
	}

	// Run service
	writeLog("Starting service...")
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}

func writeLog(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)

	// Write to file
	f, err := os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write log: %v\n", err)
		return
	}
	defer f.Close()
	f.WriteString(logLine)

	// Also log via service logger if available
	if logger != nil {
		logger.Info(message)
	} else {
		fmt.Print(logLine)
	}
}

// cleanupVSS removes all orphaned VSS snapshots at service startup
func cleanupVSS() error {
	// List all shadows first
	listCmd := exec.Command("vssadmin", "list", "shadows")
	output, err := listCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list VSS shadows: %w", err)
	}

	// Only delete if shadows exist
	if len(output) > 0 && !strings.Contains(string(output), "No items found") {
		writeLog("VSS Cleanup: Removing orphaned shadow copies...")
		deleteCmd := exec.Command("vssadmin", "delete", "shadows", "/all", "/quiet")
		_, err := deleteCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("VSS cleanup failed: %w", err)
		}
		writeLog("VSS Cleanup: Successfully removed orphaned snapshots")
	} else {
		writeLog("VSS Cleanup: No orphaned snapshots found")
	}

	return nil
}
