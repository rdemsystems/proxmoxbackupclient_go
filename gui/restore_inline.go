package main

import (
	"fmt"
	"os"
	"path/filepath"
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
func ListSnapshotsInline(baseURL, authID, secret, datastore, namespace, certFingerprint, backupID string) ([]SnapshotInfo, error) {
	writeDebugLog(fmt.Sprintf("Listing snapshots for backup ID: %s", backupID))

	// Create PBS client
	client := &pbscommon.PBSClient{
		BaseURL:         baseURL,
		CertFingerPrint: certFingerprint,
		AuthID:          authID,
		Secret:          secret,
		Datastore:       datastore,
		Namespace:       namespace,
		Insecure:        certFingerprint == "",
	}

	// List snapshots via PBS API
	client.Connect(true, "host")
	manifests, err := client.ListSnapshots()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Failed to list snapshots: %v", err))
		return nil, fmt.Errorf("Failed to list snapshots: %v", err)
	}

	result := make([]SnapshotInfo, 0)
	for _, m := range manifests {
		// Filter by backup ID if specified
		if backupID != "" && m.BackupID != backupID {
			continue
		}

		info := SnapshotInfo{
			BackupType: m.BackupType,
			BackupID:   m.BackupID,
			BackupTime: m.BackupTime,
			Size:       0, // Size not directly available from manifest
			Files:      make([]string, 0),
		}

		// Collect file names from manifest
		for _, f := range m.Files {
			info.Files = append(info.Files, f.Filename)
		}

		result = append(result, info)
	}

	writeDebugLog(fmt.Sprintf("Found %d snapshots", len(result)))
	return result, nil
}

// RestoreSnapshotInline restores a snapshot from PBS
func RestoreSnapshotInline(opts RestoreOptions) error {
	writeDebugLog(fmt.Sprintf("Starting restore: snapshot=%s, dest=%s",
		opts.SnapshotTime.Format("2006-01-02T15:04:05Z"), opts.DestPath))

	// Progress callback wrapper
	progress := func(pct float64, msg string) {
		writeDebugLog(fmt.Sprintf("Restore progress: %.1f%% - %s", pct*100, msg))
		if opts.OnProgress != nil {
			opts.OnProgress(pct, msg)
		}
	}

	// Validate options
	if opts.BaseURL == "" || opts.AuthID == "" || opts.Secret == "" {
		return fmt.Errorf("PBS connection parameters required")
	}

	if opts.BackupID == "" {
		return fmt.Errorf("Backup ID required")
	}

	if opts.DestPath == "" {
		return fmt.Errorf("Destination path required")
	}

	progress(0.05, "Connecting to PBS...")

	// Create PBS client
	client := &pbscommon.PBSClient{
		BaseURL:         opts.BaseURL,
		CertFingerPrint: opts.CertFingerprint,
		AuthID:          opts.AuthID,
		Secret:          opts.Secret,
		Datastore:       opts.Datastore,
		Namespace:       opts.Namespace,
		Insecure:        opts.CertFingerprint == "",
		Manifest: pbscommon.BackupManifest{
			BackupID:   opts.BackupID,
			BackupTime: opts.SnapshotTime,
		},
	}

	client.Connect(true, "host")
	progress(0.10, "Connected to PBS")

	// Download PXAR archive
	progress(0.20, "Downloading backup archive...")

	pxarData, err := client.DownloadToBytes("backup.pxar.didx")
	if err != nil {
		writeDebugLog(fmt.Sprintf("Failed to download PXAR: %v", err))
		return fmt.Errorf("Failed to download backup archive: %v", err)
	}

	writeDebugLog(fmt.Sprintf("Downloaded %d bytes", len(pxarData)))
	progress(0.80, "Archive downloaded")

	// Save PXAR file
	progress(0.85, "Saving archive...")

	// Create destination directory if it doesn't exist
	err = os.MkdirAll(opts.DestPath, 0755)
	if err != nil {
		return fmt.Errorf("Failed to create destination directory: %v", err)
	}

	pxarFile := filepath.Join(opts.DestPath, "backup.pxar")
	err = os.WriteFile(pxarFile, pxarData, 0644)
	if err != nil {
		return fmt.Errorf("Failed to save PXAR file: %v", err)
	}

	writeDebugLog(fmt.Sprintf("Saved PXAR file to: %s", pxarFile))
	progress(0.95, "Archive saved")

	// Note: Full PXAR extraction would require implementing the PXAR format parser
	// For now, we save the raw PXAR file which can be extracted using proxmox tools
	// or by implementing a PXAR reader in the future

	progress(1.0, "Restore completed")

	return nil
}
