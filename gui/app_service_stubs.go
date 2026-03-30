//go:build service
// +build service

// Stubs for service compilation
// These methods are required by api.BackupHandler interface
// Full implementations are in main.go (GUI mode)

package main

import (
	"fmt"
	"os"
)

// GetConfigWithHostname returns the configuration with hostname
func (a *App) GetConfigWithHostname() map[string]interface{} {
	hostname, _ := os.Hostname()
	result := map[string]interface{}{
		"hostname": hostname,
	}

	if a.config != nil {
		result["baseurl"] = a.config.BaseURL
		result["datastore"] = a.config.Datastore
		result["certfingerprint"] = a.config.CertFingerprint
		result["backup-id"] = a.config.BackupID
	}

	return result
}

// StartBackup starts a backup job
// Service implementation using RunBackupInline
func (a *App) StartBackup(backupType string, backupDirs, driveLetters, excludeList []string, backupID string, useVSS bool, compression string) error {
	writeDebugLog(fmt.Sprintf("[Service] StartBackup called: type=%s, dirs=%v, id=%s, vss=%v, compression=%s", backupType, backupDirs, backupID, useVSS, compression))

	if a.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Use hostname as fallback if backupID is empty
	if backupID == "" {
		backupID, _ = os.Hostname()
		writeDebugLog(fmt.Sprintf("[Backup ID] Empty backup-id, using hostname: %s", backupID))
	}

	// Default to "fastest" if compression is empty
	if compression == "" {
		compression = "fastest"
		writeDebugLog("[Compression] Using default: fastest")
	}

	// Merge directories: backupDirs for directory backup, driveLetters for machine backup
	var allDirs []string
	if backupType == "directory" {
		allDirs = backupDirs
	} else if backupType == "machine" {
		allDirs = driveLetters
	}

	// Prepare backup options
	opts := BackupOptions{
		BaseURL:         a.config.BaseURL,
		AuthID:          a.config.AuthID,
		Secret:          a.config.Secret,
		Datastore:       a.config.Datastore,
		Namespace:       a.config.Namespace,
		CertFingerprint: a.config.CertFingerprint,
		BackupDirs:      allDirs,
		BackupID:        backupID,
		BackupType:      backupType,
		UseVSS:          useVSS,
		Compression:     compression,
		OnProgress: func(percent float64, message string) {
			writeDebugLog(fmt.Sprintf("[Backup Progress] %.1f%% - %s", percent, message))
		},
		OnComplete: func(success bool, message string) {
			if success {
				writeDebugLog(fmt.Sprintf("[Backup Complete] SUCCESS - %s", message))
			} else {
				writeDebugLog(fmt.Sprintf("[Backup Complete] FAILED - %s", message))
			}
		},
	}

	// Execute backup using inline implementation
	writeDebugLog("[Service] Executing backup via RunBackupInline")
	return RunBackupInline(opts)
}
