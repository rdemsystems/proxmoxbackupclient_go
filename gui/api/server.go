package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Server handles HTTP API requests from the GUI
type Server struct {
	addr            string
	app             BackupHandler
	mux             *http.ServeMux
	backupProgress  map[string]*BackupProgress
	progressMutex   sync.RWMutex
}

// BackupHandler interface that the service must implement
// NOTE: StartBackup will be called in a goroutine (async), so it must be thread-safe
// TODO: Add progress callback parameters to get real-time progress updates
type BackupHandler interface {
	StartBackup(backupType string, backupDirs, driveLetters, excludeList []string, backupID string, useVSS bool) error
	GetConfigWithHostname() map[string]interface{}
	GetScheduledJobsForAPI() []map[string]interface{}
}

// NewServer creates a new API server
func NewServer(addr string, handler BackupHandler) *Server {
	s := &Server{
		addr:           addr,
		app:            handler,
		mux:            http.NewServeMux(),
		backupProgress: make(map[string]*BackupProgress),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/status", s.handleStatus)
	s.mux.HandleFunc("/backup", s.handleBackup)
	s.mux.HandleFunc("/backup/status/", s.handleBackupStatus)
	s.mux.HandleFunc("/jobs", s.handleJobs)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting API server on %s", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := s.app.GetConfigWithHostname()

	status := StatusResponse{
		Running:       true,
		Version:       "0.1.67", // TODO: get from build
		ActiveJobs:    0,         // TODO: track active jobs
		Configuration: config,
	}

	s.writeJSON(w, status, http.StatusOK)
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.BackupID == "" {
		s.writeError(w, "backup_id is required", http.StatusBadRequest)
		return
	}

	// Start backup asynchronously (don't block HTTP request)
	jobID := fmt.Sprintf("backup-%d", time.Now().Unix())

	// Initialize progress tracking
	s.progressMutex.Lock()
	s.backupProgress[jobID] = &BackupProgress{
		JobID:     jobID,
		Running:   true,
		Progress:  0,
		Message:   "Starting backup...",
		StartTime: time.Now().Format(time.RFC3339),
	}
	s.progressMutex.Unlock()

	go func() {
		log.Printf("[API] Starting async backup: %s", jobID)
		// Call startBackupDirect directly - service must execute, not route
		// Note: startBackupDirect is not exported, so we need to expose it or use a different approach
		// For now, we'll call StartBackup but ensure service App is in standalone mode
		err := s.app.StartBackup(
			req.BackupType,
			req.BackupDirs,
			req.DriveLetters,
			req.ExcludeList,
			req.BackupID,
			req.UseVSS,
		)

		// Update final status
		s.progressMutex.Lock()
		if progress, exists := s.backupProgress[jobID]; exists {
			progress.Running = false
			progress.Complete = true
			if err != nil {
				progress.Success = false
				progress.Error = err.Error()
				progress.Message = fmt.Sprintf("Backup failed: %v", err)
				log.Printf("[API] Backup %s failed: %v", jobID, err)
			} else {
				progress.Success = true
				progress.Progress = 100
				progress.Message = "Backup completed successfully"
				log.Printf("[API] Backup %s completed successfully", jobID)
			}
		}
		s.progressMutex.Unlock()
	}()

	// Return immediately with job ID
	resp := BackupResponse{
		Success: true,
		Message: "Backup started successfully (running in background)",
		JobID:   jobID,
	}

	s.writeJSON(w, resp, http.StatusOK)
}

func (s *Server) handleBackupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from URL path: /backup/status/{jobID}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/backup/status/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		s.writeError(w, "Job ID required", http.StatusBadRequest)
		return
	}
	jobID := pathParts[0]

	s.progressMutex.RLock()
	progress, exists := s.backupProgress[jobID]
	s.progressMutex.RUnlock()

	if !exists {
		s.writeError(w, "Job not found", http.StatusNotFound)
		return
	}

	s.writeJSON(w, progress, http.StatusOK)
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobsData := s.app.GetScheduledJobsForAPI()

	jobs := make([]JobInfo, 0, len(jobsData))
	for _, j := range jobsData {
		job := JobInfo{
			ID:         fmt.Sprintf("%v", j["id"]),
			Name:       fmt.Sprintf("%v", j["name"]),
			BackupType: fmt.Sprintf("%v", j["backup_type"]),
			Schedule:   fmt.Sprintf("%v", j["schedule"]),
			Status:     "idle", // TODO: track actual status
		}
		if lastRun, ok := j["last_run"].(string); ok {
			job.LastRun = lastRun
		}
		if nextRun, ok := j["next_run"].(string); ok {
			job.NextRun = nextRun
		}
		jobs = append(jobs, job)
	}

	resp := JobsResponse{Jobs: jobs}
	s.writeJSON(w, resp, http.StatusOK)
}

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, message string, status int) {
	errResp := ErrorResponse{
		Error: message,
		Code:  status,
	}
	s.writeJSON(w, errResp, status)
}
