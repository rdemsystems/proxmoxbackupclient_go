package main

import (
	"fmt"
)

// AnalyzeBackup analyzes backup directories and determines if split is needed
// Returns analysis with total size, folder breakdown, and split recommendation
func (a *App) AnalyzeBackup(backupDirs []string) (map[string]interface{}, error) {
	writeDebugLog(fmt.Sprintf("AnalyzeBackup called for %d directories", len(backupDirs)))

	analysis, err := AnalyzeBackupDirs(backupDirs)
	if err != nil {
		writeDebugLog(fmt.Sprintf("AnalyzeBackup failed: %v", err))
		return nil, err
	}

	// Convert to map for JSON serialization to frontend
	result := map[string]interface{}{
		"total_size":      analysis.TotalSize,
		"total_size_fmt":  FormatSize(analysis.TotalSize),
		"should_split":    analysis.ShouldSplit,
		"suggested_jobs":  analysis.SuggestedJobs,
		"split_threshold": SplitThreshold,
		"folders":         make([]map[string]interface{}, len(analysis.Folders)),
	}

	for i, folder := range analysis.Folders {
		result["folders"].([]map[string]interface{})[i] = map[string]interface{}{
			"path":     folder.Path,
			"name":     folder.Name,
			"size":     folder.Size,
			"size_fmt": FormatSize(folder.Size),
		}
	}

	writeDebugLog(fmt.Sprintf("Analysis: %s total, split=%v, %d jobs suggested",
		FormatSize(analysis.TotalSize), analysis.ShouldSplit, analysis.SuggestedJobs))

	return result, nil
}

// CreateBackupSplitPlan creates a plan for splitting a large backup
// Returns the split jobs that will be created
func (a *App) CreateBackupSplitPlan(backupDirs []string, backupID string) ([]map[string]interface{}, error) {
	writeDebugLog(fmt.Sprintf("CreateBackupSplitPlan called for backup ID: %s", backupID))

	analysis, err := AnalyzeBackupDirs(backupDirs)
	if err != nil {
		return nil, err
	}

	splitJobs := CreateSplitJobs(analysis, backupID)

	// Convert to map array for JSON
	result := make([]map[string]interface{}, len(splitJobs))
	for i, job := range splitJobs {
		result[i] = map[string]interface{}{
			"index":       job.Index,
			"total_jobs":  job.TotalJobs,
			"folders":     job.Folders,
			"total_size":  job.TotalSize,
			"size_fmt":    FormatSize(job.TotalSize),
			"backup_id":   job.BackupID,
			"parent_id":   job.ParentID,
		}
	}

	writeDebugLog(fmt.Sprintf("Split plan created: %d jobs", len(splitJobs)))
	return result, nil
}
