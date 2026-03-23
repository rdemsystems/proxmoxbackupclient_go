package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	// SplitThreshold: If total backup size > 100GB, propose split
	SplitThreshold = 100 * 1024 * 1024 * 1024 // 100 GB

	// MaxChunkSize: Each split job should be ~100GB max
	MaxChunkSize = 100 * 1024 * 1024 * 1024 // 100 GB
)

// FolderInfo represents a top-level folder with its size
type FolderInfo struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size uint64 `json:"size"`
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
				size := calculateDirSize(folderPath)

				analysis.Folders = append(analysis.Folders, FolderInfo{
					Path: folderPath,
					Name: entry.Name(),
					Size: size,
				})

				analysis.TotalSize += size
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

// CreateSplitJobs creates multiple smaller jobs from a large backup
// Uses bin-packing algorithm to distribute folders evenly
func CreateSplitJobs(analysis *BackupAnalysis, baseBackupID string) []SplitJob {
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

	// Bin-packing: Distribute folders into jobs of ~MaxChunkSize each
	jobs := make([]SplitJob, 0)
	currentJob := SplitJob{
		Folders:  make([]string, 0),
		ParentID: baseBackupID,
	}
	currentSize := uint64(0)

	for _, folder := range analysis.Folders {
		// If adding this folder exceeds MaxChunkSize and we already have folders, start new job
		if currentSize+folder.Size > MaxChunkSize && len(currentJob.Folders) > 0 {
			jobs = append(jobs, currentJob)
			currentJob = SplitJob{
				Folders:  make([]string, 0),
				ParentID: baseBackupID,
			}
			currentSize = 0
		}

		currentJob.Folders = append(currentJob.Folders, folder.Path)
		currentSize += folder.Size
		currentJob.TotalSize = currentSize
	}

	// Add last job if it has folders
	if len(currentJob.Folders) > 0 {
		jobs = append(jobs, currentJob)
	}

	// Set indices and backup IDs
	totalJobs := len(jobs)
	for i := range jobs {
		jobs[i].Index = i + 1
		jobs[i].TotalJobs = totalJobs
		jobs[i].BackupID = fmt.Sprintf("%s-split-%d-of-%d", baseBackupID, i+1, totalJobs)
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
