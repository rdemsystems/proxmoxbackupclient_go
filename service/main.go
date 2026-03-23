package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kardianos/service"
)

var logger service.Logger

const (
	serviceName        = "NimbusBackupService"
	serviceDisplayName = "Nimbus Backup Service"
	serviceDescription = "Executes scheduled backups to Proxmox Backup Server"
)

// Program implements the service.Interface
type program struct {
	exit           chan struct{}
	jobManager     *JobManager
	backupExecutor *BackupExecutor
}

func (p *program) Start(s service.Service) error {
	if service.Interactive() {
		logger.Info("Running in terminal mode")
	} else {
		logger.Info("Running as service")
	}
	p.exit = make(chan struct{})

	// Start service in background
	go p.run()
	return nil
}

func (p *program) run() {
	logger.Infof("Service %s started", serviceName)

	// Clean up orphaned VSS snapshots from previous crashes
	logger.Info("Running VSS cleanup...")
	if err := cleanupVSS(); err != nil {
		logger.Warningf("VSS cleanup failed (non-fatal): %v", err)
	}

	// Main service loop
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check for scheduled jobs and execute them
			p.checkAndExecuteJobs()
		case <-p.exit:
			logger.Info("Service stopping...")
			return
		}
	}
}

func (p *program) checkAndExecuteJobs() {
	logger.Info("Checking for scheduled jobs...")

	// Reload jobs from file (in case they were updated)
	if err := p.jobManager.Load(); err != nil {
		logger.Errorf("Failed to reload jobs: %v", err)
		return
	}

	// Get jobs that are due now
	dueJobs := p.jobManager.GetJobsDueNow()
	if len(dueJobs) == 0 {
		return
	}

	logger.Infof("Found %d job(s) due to run", len(dueJobs))

	// Execute each due job
	for _, job := range dueJobs {
		if err := p.backupExecutor.Execute(job); err != nil {
			logger.Errorf("Backup job failed: %s - %v", job.Name, err)
		} else {
			// Update last run time and calculate next run
			now := time.Now()
			nextRun := calculateNextRun(job.Schedule, now)
			if err := p.jobManager.UpdateJobRun(job.ID, now, nextRun); err != nil {
				logger.Errorf("Failed to update job run time: %v", err)
			}
		}
	}
}

// calculateNextRun calculates the next run time based on cron schedule
// For now, simple implementation - run every 24 hours
func calculateNextRun(schedule string, from time.Time) time.Time {
	// TODO: Parse cron schedule properly
	// For now, just add 24 hours
	return from.Add(24 * time.Hour)
}

func (p *program) Stop(s service.Service) error {
	logger.Info("Stopping service...")
	close(p.exit)
	return nil
}

func main() {
	// Command-line flags
	svcFlag := flag.String("service", "", "Control the system service: install, uninstall, start, stop, restart")
	flag.Parse()

	// Service configuration
	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
	}

	// Initialize job manager and backup executor
	jobManager, err := NewJobManager()
	if err != nil {
		log.Fatalf("Failed to create job manager: %v", err)
	}

	backupExecutor, err := NewBackupExecutor()
	if err != nil {
		log.Fatalf("Failed to create backup executor: %v", err)
	}

	prg := &program{
		jobManager:     jobManager,
		backupExecutor: backupExecutor,
	}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Setup logger
	errs := make(chan error, 5)
	logger, err = s.Logger(errs)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			err := <-errs
			if err != nil {
				log.Print(err)
			}
		}
	}()

	// Handle service control commands
	if len(*svcFlag) != 0 {
		err := service.Control(s, *svcFlag)
		if err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
		return
	}

	// Run service
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
