package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Job represents a scheduled backup job
type Job struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	BackupDirs  []string  `json:"backup_dirs"`
	Schedule    string    `json:"schedule"` // cron format: "0 2 * * *"
	Enabled     bool      `json:"enabled"`
	LastRun     time.Time `json:"last_run"`
	NextRun     time.Time `json:"next_run"`
	BackupID    string    `json:"backup_id"`
	UseVSS      bool      `json:"use_vss"`
	ExcludeList []string  `json:"exclude_list,omitempty"`
}

// JobManager handles loading and managing jobs
type JobManager struct {
	configPath string
	jobs       []Job
}

func NewJobManager() (*JobManager, error) {
	// Get config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".proxmox-backup-guardian")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	jm := &JobManager{
		configPath: filepath.Join(configDir, "jobs.json"),
		jobs:       []Job{},
	}

	// Load existing jobs
	if err := jm.Load(); err != nil {
		logger.Warningf("Failed to load jobs: %v", err)
	}

	return jm, nil
}

// Load reads jobs from config file
func (jm *JobManager) Load() error {
	data, err := os.ReadFile(jm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No jobs yet
		}
		return fmt.Errorf("failed to read jobs file: %w", err)
	}

	if err := json.Unmarshal(data, &jm.jobs); err != nil {
		return fmt.Errorf("failed to parse jobs: %w", err)
	}

	return nil
}

// GetJobsDueNow returns jobs that should run now
func (jm *JobManager) GetJobsDueNow() []Job {
	now := time.Now()
	var dueJobs []Job

	for _, job := range jm.jobs {
		if !job.Enabled {
			continue
		}

		// Check if job is due (NextRun is in the past or now)
		if job.NextRun.Before(now) || job.NextRun.Equal(now) {
			dueJobs = append(dueJobs, job)
		}
	}

	return dueJobs
}

// UpdateJobRun updates the last/next run times for a job
func (jm *JobManager) UpdateJobRun(jobID string, lastRun time.Time, nextRun time.Time) error {
	for i := range jm.jobs {
		if jm.jobs[i].ID == jobID {
			jm.jobs[i].LastRun = lastRun
			jm.jobs[i].NextRun = nextRun
			return jm.Save()
		}
	}
	return fmt.Errorf("job not found: %s", jobID)
}

// Save writes jobs to config file
func (jm *JobManager) Save() error {
	data, err := json.MarshalIndent(jm.jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	if err := os.WriteFile(jm.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write jobs file: %w", err)
	}

	return nil
}
