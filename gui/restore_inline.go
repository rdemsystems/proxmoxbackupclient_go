package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"pbscommon"
)

// RestoreOptions contains all parameters for a restore operation
type RestoreOptions struct {
	BaseURL         string
	AuthID          string
	Secret          string
	Datastore       string
	Namespace       string
	CertFingerprint string
	BackupID        string
	SnapshotTime    time.Time
	DestPath        string
	OnProgress      func(percent float64, message string)
}

// SnapshotInfo contains information about a backup snapshot
type SnapshotInfo struct {
	BackupType string
	BackupID   string
	BackupTime time.Time
	Size       int64
	Files      []string
}

// ListSnapshotsInline lists available snapshots from PBS
// SECURITY: Only lists snapshots from the specified PBS server/datastore/namespace
// This prevents cross-server snapshot access
func ListSnapshotsInline(baseURL, authID, secret, datastore, namespace, certFingerprint, backupID string) ([]SnapshotInfo, error) {
	writeBackupLog(fmt.Sprintf("Listing snapshots for backup ID: %s on %s/%s/%s", backupID, baseURL, datastore, namespace))

	// Create PBS client
	client := &pbscommon.PBSClient{
		BaseURL:          baseURL,
		CertFingerPrint:  certFingerprint,
		AuthID:           authID,
		Secret:           secret,
		Datastore:        datastore,
		Namespace:        namespace,
		Insecure:         certFingerprint != "",
		CompressionLevel: pbscommon.CompressionFastest, // Default for listing
		Manifest: pbscommon.BackupManifest{
			BackupID: backupID,
		},
	}

	// List snapshots via PBS API
	// Note: ListSnapshots() doesn't actually need Connect() for the API call
	// It uses direct HTTP GET request
	manifests, err := client.ListSnapshots()
	if err != nil {
		writeBackupLog(fmt.Sprintf("Failed to list snapshots: %v", err))
		return nil, fmt.Errorf("failed to list snapshots: %v", err)
	}

	result := make([]SnapshotInfo, 0)
	for _, m := range manifests {
		// Filter by backup ID if specified (partial match to support split backups)
		// Example: searching "JDS-SRV-1" will match "JDS-SRV-1-split-1-of-2"
		if backupID != "" && !strings.Contains(m.BackupID, backupID) {
			continue
		}

		info := SnapshotInfo{
			BackupType: m.BackupType,
			BackupID:   m.BackupID,
			BackupTime: time.Unix(m.BackupTime, 0),
			Size:       0, // Size not directly available from manifest
			Files:      make([]string, 0),
		}

		// Collect file names from manifest
		for _, f := range m.Files {
			info.Files = append(info.Files, f.Filename)
		}

		result = append(result, info)
	}

	writeBackupLog(fmt.Sprintf("Found %d snapshots", len(result)))
	return result, nil
}

// RestoreSnapshotInline restores a snapshot from PBS
// SECURITY: Only restores from the configured PBS server/datastore/namespace
// Snapshots from other servers will fail with HTTP 404
func RestoreSnapshotInline(opts RestoreOptions) error {
	writeBackupLog(fmt.Sprintf("Starting restore: snapshot=%s, dest=%s from %s/%s/%s",
		opts.SnapshotTime.Format("2006-01-02T15:04:05Z"), opts.DestPath,
		opts.BaseURL, opts.Datastore, opts.Namespace))

	// Progress callback wrapper
	progress := func(pct float64, msg string) {
		writeBackupLog(fmt.Sprintf("Restore progress: %.1f%% - %s", pct*100, msg))
		if opts.OnProgress != nil {
			opts.OnProgress(pct, msg)
		}
	}

	// Validate options
	if opts.BaseURL == "" || opts.AuthID == "" || opts.Secret == "" {
		return fmt.Errorf("PBS connection parameters required")
	}

	if opts.BackupID == "" {
		return fmt.Errorf("backup ID required")
	}

	if opts.DestPath == "" {
		return fmt.Errorf("destination path required")
	}

	// SECURITY: Verify datastore and namespace are specified
	// This prevents accidentally restoring from wrong location
	if opts.Datastore == "" {
		return fmt.Errorf("datastore required for security")
	}

	progress(0.05, "Connecting to PBS...")

	// Create PBS client
	client := &pbscommon.PBSClient{
		BaseURL:          opts.BaseURL,
		CertFingerPrint:  opts.CertFingerprint,
		AuthID:           opts.AuthID,
		Secret:           opts.Secret,
		Datastore:        opts.Datastore,
		Namespace:        opts.Namespace,
		Insecure:         opts.CertFingerprint != "",
		CompressionLevel: pbscommon.CompressionFastest, // Default for restore
		Manifest: pbscommon.BackupManifest{
			BackupID:   opts.BackupID,
			BackupTime: opts.SnapshotTime.Unix(),
		},
	}

	client.Connect(true, "host")
	progress(0.10, "Connected to PBS")

	// Download PXAR archive
	progress(0.20, "Downloading backup archive...")

	pxarData, err := client.DownloadToBytes("backup.pxar.didx")
	if err != nil {
		writeBackupLog(fmt.Sprintf("Failed to download PXAR: %v", err))
		return fmt.Errorf("failed to download backup archive: %v", err)
	}

	writeBackupLog(fmt.Sprintf("Downloaded %d bytes", len(pxarData)))
	progress(0.80, "Archive downloaded")

	// Extract PXAR archive (BETA - basic files and directories only)
	progress(0.85, "Extracting files...")

	// Create destination directory if it doesn't exist
	err = os.MkdirAll(opts.DestPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	// Parse and extract PXAR
	reader := pbscommon.NewPXARReader(pxarData)
	extracted, err := reader.ExtractAll(opts.DestPath)
	if err != nil {
		writeBackupLog(fmt.Sprintf("PXAR extraction failed: %v", err))
		return fmt.Errorf("failed to extract archive: %v", err)
	}

	// Count results
	successCount := 0
	skipCount := 0
	dirCount := 0
	for _, f := range extracted {
		if f.Skipped {
			skipCount++
			writeBackupLog(fmt.Sprintf("SKIPPED: %s - %s", f.Path, f.SkipReason))
		} else if f.IsDir {
			dirCount++
		} else {
			successCount++
		}
	}

	writeBackupLog(fmt.Sprintf("Extraction complete: %d files, %d dirs, %d skipped",
		successCount, dirCount, skipCount))
	progress(0.95, fmt.Sprintf("Extracted %d files", successCount))

	progress(1.0, "Restore completed")

	// Return warning if files were skipped
	if skipCount > 0 {
		return fmt.Errorf("restore completed with %d skipped files (see logs)", skipCount)
	}

	return nil
}
