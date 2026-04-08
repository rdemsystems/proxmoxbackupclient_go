package main

import (
	"encoding/json"
	"runtime"
	"time"
)

const BackupMetaFilename = ".nimbus_backup_meta.json"

// BackupMeta stores metadata about the backup, injected as a virtual file
// at the root of the PXAR archive. Allows restore tools to recover the
// original path (with spaces, accents) from the sanitized backup-id.
type BackupMeta struct {
	BackupID      string `json:"backup_id"`
	OriginalPath  string `json:"original_path"`
	Hostname      string `json:"hostname"`
	BackupTime    string `json:"backup_time"`
	ClientVersion string `json:"client_version"`
	OS            string `json:"os"`
	VSSUsed       bool   `json:"vss_used"`
}

func GenerateBackupMeta(backupID, originalPath, hostname string, vssUsed bool) ([]byte, error) {
	meta := BackupMeta{
		BackupID:      backupID,
		OriginalPath:  originalPath,
		Hostname:      hostname,
		BackupTime:    time.Now().UTC().Format(time.RFC3339),
		ClientVersion: appVersion,
		OS:            runtime.GOOS,
		VSSUsed:       vssUsed,
	}
	return json.MarshalIndent(meta, "", "  ")
}
