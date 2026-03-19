package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents PBS configuration
type Config struct {
	BaseURL         string `json:"baseurl"`
	CertFingerprint string `json:"certfingerprint"`
	AuthID          string `json:"authid"`
	Secret          string `json:"secret"`
	Datastore       string `json:"datastore"`
	Namespace       string `json:"namespace"`
	BackupID        string `json:"backup-id"`
	UseVSS          bool   `json:"usevss"`
}

// BackupExecutor handles backup execution
type BackupExecutor struct {
	config *Config
}

func NewBackupExecutor() (*BackupExecutor, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &BackupExecutor{
		config: config,
	}, nil
}

// Execute runs a backup job
func (be *BackupExecutor) Execute(job Job) error {
	logger.Infof("Starting backup job: %s", job.Name)

	// TODO: Implement actual backup using pbscommon
	// For now, just log
	logger.Infof("Backing up directories: %v", job.BackupDirs)
	logger.Infof("Backup ID: %s, VSS: %v", job.BackupID, job.UseVSS)

	// Simulate backup duration
	time.Sleep(2 * time.Second)

	logger.Infof("Backup job completed: %s", job.Name)
	return nil
}

func loadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".proxmox-backup-guardian", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}
