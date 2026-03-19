// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kardianos/service"
	"github.com/tizbac/proxmoxbackupclient_go/gui/api"
)

// NimbusService wraps the application for Windows Service execution
type NimbusService struct {
	app       *App
	apiServer *api.Server
}

// Start is called when the service starts
func (s *NimbusService) Start(svc service.Service) error {
	writeDebugLog("NimbusBackup service starting...")
	go s.run()
	return nil
}

// run contains the main service loop
func (s *NimbusService) run() {
	writeDebugLog("NimbusBackup service running")

	// Initialize app
	s.app = &App{
		config:        LoadConfig(),
		stopScheduler: make(chan struct{}),
	}

	// Load configuration (service will read config from file when needed)
	configMap := s.app.GetConfigWithHostname()
	if hostname, ok := configMap["hostname"].(string); ok {
		writeDebugLog(fmt.Sprintf("Service: Running for %s", hostname))
	} else {
		writeDebugLog("Service: Running in background")
	}

	// Config will be loaded from file by each scheduled job when needed

	// Clean up any abandoned jobs from previous crash
	s.app.CleanupAbandonedJobs()

	// Start the scheduler
	s.app.StartScheduler()

	// Start HTTP API server for GUI communication
	s.apiServer = api.NewServer("127.0.0.1:18765", s.app)
	writeDebugLog("Starting HTTP API server on 127.0.0.1:18765")

	go func() {
		if err := s.apiServer.Start(); err != nil {
			writeDebugLog(fmt.Sprintf("API server error: %v", err))
		}
	}()

	// Keep the service running (scheduler runs in background goroutine)
	for {
		time.Sleep(1 * time.Minute)
		// Service is alive, scheduler and API server run in background
		// When Stop() is called, the scheduler will stop and this loop will be interrupted
	}
}

// Stop is called when the service stops
func (s *NimbusService) Stop(svc service.Service) error {
	writeDebugLog("NimbusBackup service stopping...")

	// Stop the scheduler gracefully
	if s.app != nil {
		s.app.StopScheduler()
	}

	// Give it a moment to finish current operations
	time.Sleep(2 * time.Second)

	writeDebugLog("NimbusBackup service stopped")
	return nil
}

// RunAsService starts the application as a Windows Service
func RunAsService() {
	writeDebugLog("Attempting to run as Windows Service")

	svcConfig := &service.Config{
		Name:        "NimbusBackup",
		DisplayName: "Nimbus Backup Service",
		Description: "Executes scheduled backups to Proxmox Backup Server with VSS support",
	}

	nimbusSvc := &NimbusService{}
	s, err := service.New(nimbusSvc, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	logger, err := s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}

	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}

// IsServiceMode checks if running in service mode
func IsServiceMode() bool {
	for _, arg := range os.Args {
		if arg == "--service" {
			return true
		}
	}
	return false
}
