package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ScheduledJob represents a scheduled backup job
type ScheduledJob struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	ScheduleTime string   `json:"scheduleTime"` // HH:MM format
	RunAtStartup bool     `json:"runAtStartup"`
	BackupDirs   []string `json:"backupDirs"`
	BackupID     string   `json:"backupId"`
	UseVSS       bool     `json:"useVSS"`
	BackupType   string   `json:"backupType"`
	ExcludeList  []string `json:"excludeList"`
	LastRun      string   `json:"lastRun,omitempty"`  // ISO timestamp
	NextRun      string   `json:"nextRun,omitempty"`  // ISO timestamp
	Enabled      bool     `json:"enabled"`
}

// JobHistory represents a completed backup job
type JobHistory struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Timestamp  string   `json:"timestamp"` // ISO format
	Status     string   `json:"status"`    // "success", "failed", "running"
	Message    string   `json:"message"`
	BackupDirs []string `json:"backupDirs"`
	BackupID   string   `json:"backupId"`
	UseVSS     bool     `json:"useVSS"`
}

func getScheduledJobsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".proxmox-backup-guardian")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "scheduled_jobs.json"), nil
}

func getJobHistoryPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".proxmox-backup-guardian")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "job_history.json"), nil
}

// SaveScheduledJob saves a new scheduled job
func (a *App) SaveScheduledJob(job ScheduledJob) error {
	writeDebugLog(fmt.Sprintf("SaveScheduledJob called for: %s", job.Name))

	// Load existing jobs
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Error loading existing jobs: %v", err))
	}

	// Set enabled by default
	job.Enabled = true

	// Calculate next run
	job.NextRun = calculateNextRun(job.ScheduleTime)

	// Add new job
	jobs = append(jobs, job)

	// Save to file
	jobsPath, err := getScheduledJobsPath()
	if err != nil {
		return fmt.Errorf("failed to get jobs path: %w", err)
	}

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	if err := os.WriteFile(jobsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write jobs file: %w", err)
	}

	writeDebugLog(fmt.Sprintf("Scheduled job saved: %s (next run: %s)", job.Name, job.NextRun))

	// If this job has runAtStartup enabled, ensure app auto-start is enabled
	if job.RunAtStartup {
		writeDebugLog("Job has runAtStartup=true, enabling system auto-start")
		if err := a.EnableAutoStart(); err != nil {
			writeDebugLog(fmt.Sprintf("Warning: Failed to enable auto-start: %v", err))
			// Don't fail the whole operation if auto-start fails
		}
	}

	return nil
}

// GetScheduledJobs returns all scheduled jobs
func (a *App) GetScheduledJobs() ([]ScheduledJob, error) {
	jobsPath, err := getScheduledJobsPath()
	if err != nil {
		return []ScheduledJob{}, err
	}

	data, err := os.ReadFile(jobsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScheduledJob{}, nil // No jobs yet
		}
		return nil, err
	}

	var jobs []ScheduledJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

// DeleteScheduledJob removes a scheduled job by ID
func (a *App) DeleteScheduledJob(jobID string) error {
	writeDebugLog(fmt.Sprintf("DeleteScheduledJob called for ID: %s", jobID))

	jobs, err := a.GetScheduledJobs()
	if err != nil {
		return err
	}

	// Filter out the job to delete
	filtered := []ScheduledJob{}
	for _, job := range jobs {
		if job.ID != jobID {
			filtered = append(filtered, job)
		}
	}

	// Save updated list
	jobsPath, err := getScheduledJobsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(jobsPath, data, 0600)
}

// GetJobHistory returns job history
func (a *App) GetJobHistory() ([]JobHistory, error) {
	historyPath, err := getJobHistoryPath()
	if err != nil {
		return []JobHistory{}, err
	}

	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []JobHistory{}, nil
		}
		return nil, err
	}

	var history []JobHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return history, nil
}

// AddJobHistory adds a job to history
func (a *App) AddJobHistory(entry JobHistory) error {
	history, err := a.GetJobHistory()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Error loading history: %v", err))
	}

	// Add new entry at the beginning
	history = append([]JobHistory{entry}, history...)

	// Keep only last 50 entries
	if len(history) > 50 {
		history = history[:50]
	}

	// Save
	historyPath, err := getJobHistoryPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(historyPath, data, 0600)
}

// calculateNextRun calculates the next run time based on schedule time (HH:MM)
func calculateNextRun(scheduleTime string) string {
	parts := strings.Split(scheduleTime, ":")
	if len(parts) != 2 {
		return ""
	}

	now := time.Now()
	var hour, min int
	if _, err := fmt.Sscanf(scheduleTime, "%d:%d", &hour, &min); err != nil {
		writeDebugLog(fmt.Sprintf("Error parsing schedule time %s: %v", scheduleTime, err))
		return ""
	}

	// Schedule for today at the specified time
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location())

	// If time has already passed today, schedule for tomorrow
	if nextRun.Before(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	return nextRun.Format(time.RFC3339)
}

// StartScheduler starts the background job scheduler
func (a *App) StartScheduler() {
	writeDebugLog("Starting background job scheduler")

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			a.checkAndRunScheduledJobs()
		}
	}()
}

// HandleStartupRun executes scheduled jobs that have runAtStartup enabled
func (a *App) HandleStartupRun() {
	writeDebugLog("HandleStartupRun called - checking for startup jobs")

	jobs, err := a.GetScheduledJobs()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Error loading scheduled jobs: %v", err))
		return
	}

	for _, job := range jobs {
		if !job.Enabled || !job.RunAtStartup {
			continue
		}

		writeDebugLog(fmt.Sprintf("Executing startup job: %s", job.Name))
		go a.executeScheduledJob(job)
	}
}

// checkAndRunScheduledJobs checks if any jobs need to run
func (a *App) checkAndRunScheduledJobs() {
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Error loading scheduled jobs: %v", err))
		return
	}

	now := time.Now()

	for _, job := range jobs {
		if !job.Enabled {
			continue
		}

		// Parse next run time
		if job.NextRun == "" {
			continue
		}

		nextRun, err := time.Parse(time.RFC3339, job.NextRun)
		if err != nil {
			writeDebugLog(fmt.Sprintf("Error parsing next run time: %v", err))
			continue
		}

		// Check if it's time to run (within 2 minute window to avoid missing)
		if now.After(nextRun) && now.Before(nextRun.Add(2*time.Minute)) {
			writeDebugLog(fmt.Sprintf("Executing scheduled job: %s", job.Name))
			go a.executeScheduledJob(job)
		}
	}
}

// executeScheduledJob executes a scheduled job
func (a *App) executeScheduledJob(job ScheduledJob) {
	writeDebugLog(fmt.Sprintf("Executing scheduled job: %s", job.Name))

	// Add to history as "running"
	historyEntry := JobHistory{
		ID:         fmt.Sprintf("%d", time.Now().Unix()),
		Name:       job.Name,
		Timestamp:  time.Now().Format(time.RFC3339),
		Status:     "running",
		Message:    "Backup en cours...",
		BackupDirs: job.BackupDirs,
		BackupID:   job.BackupID,
		UseVSS:     job.UseVSS,
	}
	if err := a.AddJobHistory(historyEntry); err != nil {
		writeDebugLog(fmt.Sprintf("Warning: Failed to add job history: %v", err))
	}

	// Map frontend BackupType to PBS BackupType
	pbsBackupType := "host" // Default for directory backups
	if job.BackupType == "machine" {
		pbsBackupType = "vm"
	}

	// Execute backup using RunBackupInline
	opts := BackupOptions{
		BaseURL:         a.config.BaseURL,
		AuthID:          a.config.AuthID,
		Secret:          a.config.Secret,
		Datastore:       a.config.Datastore,
		Namespace:       a.config.Namespace,
		CertFingerprint: a.config.CertFingerprint,
		BackupDirs:      job.BackupDirs,
		BackupID:        job.BackupID,
		BackupType:      pbsBackupType,
		UseVSS:          job.UseVSS,
		OnProgress: func(percent float64, message string) {
			// Log progress for scheduled backups
			writeDebugLog(fmt.Sprintf("Scheduled backup progress: %.1f%% - %s", percent, message))
		},
		OnComplete: func(success bool, message string) {
			// Update history entry
			historyEntry.Status = "success"
			historyEntry.Message = message
			if !success {
				historyEntry.Status = "failed"
			}
			historyEntry.Timestamp = time.Now().Format(time.RFC3339)
			if err := a.AddJobHistory(historyEntry); err != nil {
				writeDebugLog(fmt.Sprintf("Warning: Failed to add job history: %v", err))
			}

			writeDebugLog(fmt.Sprintf("Scheduled job completed: %s - success=%v", job.Name, success))
		},
	}

	// Run the backup
	err := RunBackupInline(opts)
	if err != nil {
		writeDebugLog(fmt.Sprintf("Scheduled job error: %v", err))
		historyEntry.Status = "failed"
		historyEntry.Message = fmt.Sprintf("Erreur: %v", err)
		historyEntry.Timestamp = time.Now().Format(time.RFC3339)
		if err := a.AddJobHistory(historyEntry); err != nil {
			writeDebugLog(fmt.Sprintf("Warning: Failed to add job history: %v", err))
		}
	}

	// Update job's last run and calculate next run
	jobs, _ := a.GetScheduledJobs()
	for i, j := range jobs {
		if j.ID == job.ID {
			jobs[i].LastRun = time.Now().Format(time.RFC3339)
			jobs[i].NextRun = calculateNextRun(job.ScheduleTime)
			break
		}
	}

	// Save updated jobs
	jobsPath, _ := getScheduledJobsPath()
	data, _ := json.MarshalIndent(jobs, "", "  ")
	if err := os.WriteFile(jobsPath, data, 0600); err != nil {
		writeDebugLog(fmt.Sprintf("Warning: Failed to save updated jobs: %v", err))
	}
}

