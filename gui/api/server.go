package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Server handles HTTP API requests from the GUI
type Server struct {
	addr string
	app  BackupHandler
	mux  *http.ServeMux
	mu   sync.Mutex
}

// BackupHandler interface that the service must implement
// NOTE: StartBackup will be called in a goroutine (async), so it must be thread-safe
type BackupHandler interface {
	StartBackup(backupType, backupID string, backupDirs, driveLetters []string, useVSS bool) error
	GetConfigWithHostname() map[string]interface{}
	GetScheduledJobs() []map[string]interface{}
}

// NewServer creates a new API server
func NewServer(addr string, handler BackupHandler) *Server {
	s := &Server{
		addr: addr,
		app:  handler,
		mux:  http.NewServeMux(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/status", s.handleStatus)
	s.mux.HandleFunc("/backup", s.handleBackup)
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
		Version:       "0.1.58", // TODO: get from build
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

	go func() {
		writeDebugLog(fmt.Sprintf("Starting async backup: %s", jobID))
		err := s.app.StartBackup(
			req.BackupType,
			req.BackupID,
			req.BackupDirs,
			req.DriveLetters,
			req.UseVSS,
		)
		if err != nil {
			writeDebugLog(fmt.Sprintf("Backup %s failed: %v", jobID, err))
		} else {
			writeDebugLog(fmt.Sprintf("Backup %s completed successfully", jobID))
		}
	}()

	// Return immediately with job ID
	resp := BackupResponse{
		Success: true,
		Message: "Backup started successfully (running in background)",
		JobID:   jobID,
	}

	s.writeJSON(w, resp, http.StatusOK)
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobsData := s.app.GetScheduledJobs()

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
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, message string, status int) {
	errResp := ErrorResponse{
		Error: message,
		Code:  status,
	}
	s.writeJSON(w, errResp, status)
}
