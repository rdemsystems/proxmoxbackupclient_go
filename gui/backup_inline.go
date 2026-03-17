package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pbscommon"
)

// BackupOptions contains all parameters for a backup operation
type BackupOptions struct {
	BaseURL         string
	AuthID          string
	Secret          string
	Datastore       string
	Namespace       string
	CertFingerprint string
	BackupDirs      []string // Multiple directories or drives to backup
	BackupID        string
	BackupType      string // "host" for directory, "vm" for machine
	UseVSS          bool
	OnProgress      func(percent float64, message string)
	OnComplete      func(success bool, message string)
}

// RunBackupInline performs a backup without external binaries
func RunBackupInline(opts BackupOptions) error {
	writeDebugLog("Starting inline backup")

	// Validate options
	if opts.BaseURL == "" || opts.AuthID == "" || opts.Secret == "" {
		return fmt.Errorf("PBS connection parameters required")
	}

	if len(opts.BackupDirs) == 0 {
		return fmt.Errorf("At least one backup directory or drive required")
	}

	// Use hostname as backup ID if not specified
	if opts.BackupID == "" {
		hostname, err := os.Hostname()
		if err == nil {
			opts.BackupID = hostname
		} else {
			opts.BackupID = "unnamed-backup"
		}
	}

	// Default to "host" type for directory backups
	if opts.BackupType == "" {
		opts.BackupType = "host"
	}

	// Progress callback wrapper
	progress := func(pct float64, msg string) {
		writeDebugLog(fmt.Sprintf("Backup progress: %.1f%% - %s", pct*100, msg))
		if opts.OnProgress != nil {
			opts.OnProgress(pct, msg)
		}
	}

	// Create PBS client
	progress(0.05, "Connecting to PBS...")
	client := &pbscommon.PBSClient{
		BaseURL:         opts.BaseURL,
		CertFingerPrint: opts.CertFingerprint,
		AuthID:          opts.AuthID,
		Secret:          opts.Secret,
		Datastore:       opts.Datastore,
		Namespace:       opts.Namespace,
		Insecure:        opts.CertFingerprint == "",
	}

	progress(0.10, "Connected to PBS")

	// Create backup snapshot
	progress(0.15, "Creating snapshot...")
	backupTime := time.Now().Format("2006-01-02T15:04:05Z")

	// Start backup session
	writeDebugLog(fmt.Sprintf("Starting backup for %s/%s/%s", opts.BackupType, opts.BackupID, backupTime))

	// For directory backups, we need to:
	// 1. Create a PXAR archive
	// 2. Chunk it and upload to PBS
	// 3. Create the manifest

	progress(0.20, "Scanning files...")

	// Check if all backup directories exist
	for _, dir := range opts.BackupDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("Backup directory does not exist: %s", dir)
			writeDebugLog(errMsg)
			if opts.OnComplete != nil {
				opts.OnComplete(false, errMsg)
			}
			return fmt.Errorf(errMsg)
		}
	}

	// Calculate total size for progress tracking across all directories
	progress(0.25, "Calculating backup size...")
	var totalSize int64
	var err error
	for _, dir := range opts.BackupDirs {
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if !info.IsDir() {
				totalSize += info.Size()
			}
			return nil
		})
		if err != nil {
			writeDebugLog(fmt.Sprintf("Warning: failed to calculate size for %s: %v", dir, err))
		}
	}

	writeDebugLog(fmt.Sprintf("Total backup size: %d bytes", totalSize))
	progress(0.30, fmt.Sprintf("Backing up %d bytes", totalSize))

	// For now, return a message that the backup is starting
	// The full implementation would:
	// 1. Use clientcommon.CreatePXARArchive() to create the archive
	// 2. Chunk the data using pbscommon.Chunker
	// 3. Upload chunks with deduplication
	// 4. Create and upload the manifest

	// This is a simplified implementation that shows the structure
	// The full logic from directorybackup/main.go needs to be integrated here

	progress(0.50, "Uploading data to PBS...")

	// TODO: Full implementation requires:
	// - PXAR archive creation
	// - Chunking and deduplication (using pbscommon.Chunker)
	// - Dynamic chunk upload (client.UploadDynamicCompressedChunk)
	// - Index creation and management
	// - Manifest upload
	// - Proper error handling throughout

	_ = client // Will be used in full implementation

	errMsg := "Fonctionnalité de backup non encore implémentée. L'implémentation complète nécessite l'intégration du code PXAR et de l'upload vers PBS."
	writeDebugLog(errMsg)
	if opts.OnComplete != nil {
		opts.OnComplete(false, errMsg)
	}
	return fmt.Errorf(errMsg)
}
