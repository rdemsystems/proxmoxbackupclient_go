package main

import (
	"encoding/json"
	"fmt"
)

// API wrapper methods for HTTP API compatibility
// These methods convert map[string]interface{} to typed structs

// SaveScheduledJobFromMap is an API wrapper that accepts map[string]interface{}
func (a *App) SaveScheduledJobFromMap(jobData map[string]interface{}) error {
	// Convert map to JSON then unmarshal to ScheduledJob
	jsonData, err := json.Marshal(jobData)
	if err != nil {
		return fmt.Errorf("failed to marshal job data: %w", err)
	}

	var job ScheduledJob
	if err := json.Unmarshal(jsonData, &job); err != nil {
		return fmt.Errorf("failed to unmarshal job data: %w", err)
	}

	return a.SaveScheduledJob(job)
}

// UpdateScheduledJobFromMap is an API wrapper that accepts map[string]interface{}
func (a *App) UpdateScheduledJobFromMap(jobData map[string]interface{}) error {
	jsonData, err := json.Marshal(jobData)
	if err != nil {
		return fmt.Errorf("failed to marshal job data: %w", err)
	}

	var job ScheduledJob
	if err := json.Unmarshal(jsonData, &job); err != nil {
		return fmt.Errorf("failed to unmarshal job data: %w", err)
	}

	return a.UpdateScheduledJob(job)
}

// DeleteScheduledJobFromMap is an API wrapper (same signature, just for consistency)
func (a *App) DeleteScheduledJobFromMap(jobID string) error {
	return a.DeleteScheduledJob(jobID)
}
