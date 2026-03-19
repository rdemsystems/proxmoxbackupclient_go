// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kardianos/service"
)

// NimbusService wraps the application for Windows Service execution
type NimbusService struct {
	app *App
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
	s.app = &App{}

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

	// Keep the service running
	for {
		time.Sleep(1 * time.Minute)
		// Service is alive, scheduler runs in background
	}
}

// Stop is called when the service stops
func (s *NimbusService) Stop(svc service.Service) error {
	writeDebugLog("NimbusBackup service stopping...")
	// Cleanup if needed
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
