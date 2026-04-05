package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// SplitThreshold: If total backup size > 100GB, propose split
	SplitThreshold = 100 * 1024 * 1024 * 1024 // 100 GB

	// MaxChunkSize: Each split job should be ~50GB max (reduced from 100GB to prevent manifest overflow)
	// PBS has a ~4-6MB limit on manifest blob size. With 50GB jobs, we get ~15k chunks/job = ~3-4MB manifest
	MaxChunkSize = 50 * 1024 * 1024 * 1024 // 50 GB
)

// GenerateBackupID creates a backup-id from hostname and path
// Format: hostname_DRIVE_PATH (e.g., SERVER01_D_DATA_Users)
func GenerateBackupID(hostname, path string) string {
	// Clean path and replace backslashes with underscores
	cleanPath := filepath.Clean(path)
	cleanPath = strings.ReplaceAll(cleanPath, "\\", "_")
	cleanPath = strings.ReplaceAll(cleanPath, "/", "_")
	cleanPath = strings.ReplaceAll(cleanPath, ":", "")

	// Remove leading/trailing underscores
	cleanPath = strings.Trim(cleanPath, "_")

	// Combine hostname and path
	if cleanPath == "" {
		return hostname
	}
	return fmt.Sprintf("%s_%s", hostname, cleanPath)
}

// FolderInfo represents a top-level folder with its size
type FolderInfo struct {
	Path           string `json:"path"`
	Name           string `json:"name"`
	Size           uint64 `json:"size"`
	AccessDenied   bool   `json:"access_denied"`   // True if size couldn't be calculated due to permissions
	BackupExists   bool   `json:"backup_exists"`   // True if previous backup exists on PBS for this folder
	BackupID       string `json:"backup_id"`       // Individual backup-id for this folder
}

// BackupAnalysis contains the analysis of directories to backup
type BackupAnalysis struct {
	TotalSize     uint64        `json:"total_size"`
	Folders       []FolderInfo  `json:"folders"`
	ShouldSplit   bool          `json:"should_split"`
	SuggestedJobs int           `json:"suggested_jobs"`
}

// AnalyzeBackupDirs analyzes the top-level folders in the backup directories
// Returns total size and list of folders with their sizes
func AnalyzeBackupDirs(backupDirs []string) (*BackupAnalysis, error) {
	analysis := &BackupAnalysis{
		Folders: make([]FolderInfo, 0),
	}

	for _, dir := range backupDirs {
		// Check if directory exists
		info, err := os.Stat(dir)
		if err != nil {
			return nil, fmt.Errorf("cannot access directory %s: %w", dir, err)
		}

		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", dir)
		}

		// List top-level folders
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("cannot read directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				folderPath := filepath.Join(dir, entry.Name())
				size, err := calculateDirSize(folderPath)

				folderInfo := FolderInfo{
					Path: folderPath,
					Name: entry.Name(),
					Size: size,
				}

				// If access denied, mark folder and estimate large size (will be split separately)
				if err != nil && strings.Contains(err.Error(), "access denied") {
					folderInfo.AccessDenied = true
					// Estimate 500GB for denied folders (will be in separate job with VSS)
					folderInfo.Size = 500 * 1024 * 1024 * 1024
				}

				analysis.Folders = append(analysis.Folders, folderInfo)
				analysis.TotalSize += folderInfo.Size
			}
		}
	}

	// Sort folders by size (largest first) for better job distribution
	sort.Slice(analysis.Folders, func(i, j int) bool {
		return analysis.Folders[i].Size > analysis.Folders[j].Size
	})

	// Determine if split is needed
	analysis.ShouldSplit = analysis.TotalSize > SplitThreshold

	// Calculate suggested number of jobs
	if analysis.ShouldSplit {
		analysis.SuggestedJobs = int((analysis.TotalSize + MaxChunkSize - 1) / MaxChunkSize)
		if analysis.SuggestedJobs > 10 {
			analysis.SuggestedJobs = 10 // Max 10 jobs
		}
	} else {
		analysis.SuggestedJobs = 1
	}

	return analysis, nil
}

// SplitJob represents a partial backup job for a large backup
type SplitJob struct {
	Index      int      `json:"index"`
	TotalJobs  int      `json:"total_jobs"`
	Folders    []string `json:"folders"`
	TotalSize  uint64   `json:"total_size"`
	BackupID   string   `json:"backup_id"`
	ParentID   string   `json:"parent_id"` // Original job ID
}

// getSubFolders returns direct sub-folders of a path with their sizes
func getSubFolders(path string) ([]FolderInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	subFolders := make([]FolderInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			subPath := filepath.Join(path, entry.Name())
			size, err := calculateDirSize(subPath)

			subFolder := FolderInfo{
				Path: subPath,
				Name: entry.Name(),
				Size: size,
			}

			// If access denied, mark it but don't skip
			if err != nil && strings.Contains(err.Error(), "access denied") {
				subFolder.AccessDenied = true
				subFolder.Size = 10 * 1024 * 1024 * 1024 // Estimate 10GB
			}

			subFolders = append(subFolders, subFolder)
		}
	}

	// Sort by size (largest first)
	sort.Slice(subFolders, func(i, j int) bool {
		return subFolders[i].Size > subFolders[j].Size
	})

	return subFolders, nil
}

// CreateSplitJobs creates multiple smaller jobs from a large backup
// Uses bin-packing algorithm to distribute folders evenly
// Folders with AccessDenied are placed in separate jobs (will use VSS)
func CreateSplitJobs(analysis *BackupAnalysis, baseBackupID string, hostname string) []SplitJob {
	if !analysis.ShouldSplit {
		// No split needed, return single job
		allFolders := make([]string, len(analysis.Folders))
		for i, f := range analysis.Folders {
			allFolders[i] = f.Path
		}
		return []SplitJob{{
			Index:     1,
			TotalJobs: 1,
			Folders:   allFolders,
			TotalSize: analysis.TotalSize,
			BackupID:  baseBackupID,
			ParentID:  baseBackupID,
		}}
	}

	jobs := make([]SplitJob, 0)

	// Separate folders: accessible vs denied, and filter out already backed-up folders
	accessibleFolders := make([]FolderInfo, 0)
	deniedFolders := make([]FolderInfo, 0)

	for _, folder := range analysis.Folders {
		// Skip folders that already have backups (dedup will handle them efficiently)
		if folder.BackupExists {
			continue
		}

		if folder.AccessDenied {
			deniedFolders = append(deniedFolders, folder)
		} else {
			accessibleFolders = append(accessibleFolders, folder)
		}
	}

	// If no folders need splitting (all backed up or too small), return empty
	if len(accessibleFolders) == 0 && len(deniedFolders) == 0 {
		// All folders already backed up - return single job with all folders
		allFolders := make([]string, len(analysis.Folders))
		for i, f := range analysis.Folders {
			allFolders[i] = f.Path
		}
		return []SplitJob{{
			Index:     1,
			TotalJobs: 1,
			Folders:   allFolders,
			TotalSize: analysis.TotalSize,
			BackupID:  baseBackupID,
			ParentID:  baseBackupID,
		}}
	}

	// Create one job per top-level folder (both denied and accessible)
	// If a folder > MaxChunkSize, subdivide it into sub-folders
	allFolders := append(deniedFolders, accessibleFolders...)

	for _, folder := range allFolders {
		// If folder is too large, subdivide into direct sub-folders
		if folder.Size > MaxChunkSize && !folder.AccessDenied {
			subFolders, err := getSubFolders(folder.Path)
			if err == nil && len(subFolders) > 1 {
				// Create one job per sub-folder
				for _, subFolder := range subFolders {
					backupID := GenerateBackupID(hostname, subFolder.Path)
					jobs = append(jobs, SplitJob{
						Folders:   []string{subFolder.Path},
						TotalSize: subFolder.Size,
						BackupID:  backupID, // e.g., JDS-SRV-1_D_DATA_app
						ParentID:  baseBackupID,
					})
				}
				continue
			}
			// If subdivision failed or only 1 subfolder, fall through to single job
		}

		// Single job for this folder
		backupID := folder.BackupID
		if backupID == "" {
			backupID = GenerateBackupID(hostname, folder.Path)
		}
		jobs = append(jobs, SplitJob{
			Folders:   []string{folder.Path},
			TotalSize: folder.Size,
			BackupID:  backupID, // e.g., JDS-SRV-1_D_DATA, JDS-SRV-1_E_USERS
			ParentID:  baseBackupID,
		})
	}

	// Set indices
	totalJobs := len(jobs)
	for i := range jobs {
		jobs[i].Index = i + 1
		jobs[i].TotalJobs = totalJobs
	}

	return jobs
}

// FormatSize formats a size in bytes to human-readable format
func FormatSize(bytes uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}
