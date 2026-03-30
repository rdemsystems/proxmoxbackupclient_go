package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cornelk/hashmap"
	"pbscommon"
	"retry"
	"security"
	"snapshot"
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
	Compression     string   // Compression level: "fastest", "default", "better", "best"
	OnProgress      func(percent float64, message string)
	OnComplete      func(success bool, message string)
}

var didxMagic = []byte{28, 145, 78, 165, 25, 186, 179, 205}

// Global backup locks per destination (BaseURL + Datastore)
var (
	backupLocks      = make(map[string]*sync.Mutex)
	backupLocksMutex sync.Mutex
)

// getBackupLock returns a mutex for the given backup destination
func getBackupLock(baseURL, datastore string) *sync.Mutex {
	key := baseURL + "|" + datastore
	backupLocksMutex.Lock()
	defer backupLocksMutex.Unlock()

	if _, exists := backupLocks[key]; !exists {
		backupLocks[key] = &sync.Mutex{}
	}
	return backupLocks[key]
}

// calculateDirSize scans a directory recursively and returns total size in bytes
// Returns size and error if access was denied (needs VSS)
func calculateDirSize(path string) (uint64, error) {
	var totalSize uint64
	var accessDenied bool

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// Check if it's an access denied error
			if strings.Contains(err.Error(), "Access is denied") ||
			   strings.Contains(err.Error(), "permission denied") {
				accessDenied = true
				// Stop walking this path, but continue with others
				if filePath == path {
					// Root path denied - cannot scan at all
					return fmt.Errorf("access denied to root path")
				}
				return filepath.SkipDir
			}
			return nil // Skip other errors
		}
		if !info.IsDir() {
			totalSize += uint64(info.Size())
		}
		return nil
	})

	if accessDenied && totalSize == 0 {
		return 0, fmt.Errorf("access denied: %s", path)
	}

	return totalSize, err
}

type ChunkState struct {
	assignments         []string
	assignments_offset  []uint64
	pos                 uint64
	wrid                uint64
	chunkcount          uint64
	chunkdigests        hash.Hash
	current_chunk       []byte
	C                   pbscommon.Chunker
	newchunk            *atomic.Uint64
	reusechunk          *atomic.Uint64
	failedchunk         *atomic.Uint64     // Track failed chunk uploads
	knownChunks         *hashmap.Map[string, bool]
	onProgress          func(float64, string)
	lastProgressReport  uint64
	lastProgressPercent float64            // Track last reported percentage to prevent backwards progress
	totalSize           *atomic.Uint64     // Total size, updated by background scan
	uploadErrors        []string           // Collect upload errors to report at the end
	errorsMutex         sync.Mutex         // Protect uploadErrors slice
}

type DidxEntry struct {
	offset uint64
	digest []byte
}

func (c *ChunkState) Init(newchunk *atomic.Uint64, reusechunk *atomic.Uint64, failedchunk *atomic.Uint64, knownChunks *hashmap.Map[string, bool], onProgress func(float64, string), totalSize *atomic.Uint64) {
	c.assignments = make([]string, 0)
	c.assignments_offset = make([]uint64, 0)
	c.pos = 0
	c.chunkcount = 0
	c.chunkdigests = sha256.New()
	c.current_chunk = make([]byte, 0)
	c.C = pbscommon.Chunker{}
	c.C.New(1024 * 1024 * 4)
	c.reusechunk = reusechunk
	c.newchunk = newchunk
	c.failedchunk = failedchunk
	c.knownChunks = knownChunks
	c.onProgress = onProgress
	c.lastProgressReport = 0
	c.lastProgressPercent = 0.0
	c.totalSize = totalSize
	c.uploadErrors = make([]string, 0)
}

func (c *ChunkState) HandleData(b []byte, client *pbscommon.PBSClient) error {
	chunkpos := c.C.Scan(b)

	if chunkpos == 0 {
		c.current_chunk = append(c.current_chunk, b...)
	} else {
		for chunkpos > 0 {
			c.current_chunk = append(c.current_chunk, b[:chunkpos]...)

			h := sha256.New()
			if _, err := h.Write(c.current_chunk); err != nil {
				return fmt.Errorf("failed to hash chunk: %w", err)
			}
			bindigest := h.Sum(nil)
			shahash := hex.EncodeToString(bindigest)

			if _, ok := c.knownChunks.GetOrInsert(shahash, true); !ok {
				writeBackupLog(fmt.Sprintf("New chunk[%s] %d bytes", shahash, len(c.current_chunk)))

				// Retry chunk upload with exponential backoff
				chunkData := c.current_chunk // Capture for closure
				retryConfig := retry.DefaultConfig()
				retryConfig.MaxAttempts = 5 // More retries for chunk uploads
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()

				err := retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
					return client.UploadDynamicCompressedChunk(c.wrid, shahash, chunkData)
				})
				if err != nil {
					// Log error but CONTINUE processing
					errMsg := fmt.Sprintf("⚠️  Failed to upload chunk %s after %d retries: %v", shahash, retryConfig.MaxAttempts, err)
					writeBackupLog(errMsg)
					c.failedchunk.Add(1)

					// Collect error for final report
					c.errorsMutex.Lock()
					c.uploadErrors = append(c.uploadErrors, errMsg)
					c.errorsMutex.Unlock()

					// Continue with next chunk instead of failing
				} else {
					c.newchunk.Add(1)
				}
			} else {
				writeBackupLog(fmt.Sprintf("Reuse chunk[%s] %d bytes", shahash, len(c.current_chunk)))
				c.reusechunk.Add(1)
			}

			if err := binary.Write(c.chunkdigests, binary.LittleEndian, (c.pos + uint64(len(c.current_chunk)))); err != nil {
				return fmt.Errorf("failed to write chunk offset: %w", err)
			}
			if _, err := c.chunkdigests.Write(h.Sum(nil)); err != nil {
				return fmt.Errorf("failed to write chunk digest: %w", err)
			}

			c.assignments_offset = append(c.assignments_offset, c.pos)
			c.assignments = append(c.assignments, shahash)
			c.pos += uint64(len(c.current_chunk))
			c.chunkcount += 1

			// Report progress every 10 MB
			if c.onProgress != nil && c.pos-c.lastProgressReport > 10*1024*1024 {
				c.lastProgressReport = c.pos
				sizeMB := c.pos / (1024 * 1024)

				// Build progress message with chunk stats
				var msg string
				failed := c.failedchunk.Load()
				if failed > 0 {
					msg = fmt.Sprintf("Traité: %d MB (New: %d, Reused: %d, ⚠️ Failed: %d chunks)",
						sizeMB, c.newchunk.Load(), c.reusechunk.Load(), failed)
				} else {
					msg = fmt.Sprintf("Traité: %d MB (New: %d, Reused: %d chunks)",
						sizeMB, c.newchunk.Load(), c.reusechunk.Load())
				}

				// Calculate progress based on total size if available
				var progress float64
				totalSize := c.totalSize.Load()
				if totalSize > 0 {
					// Progress from 10% to 90% based on bytes processed
					progress = 0.1 + (float64(c.pos)/float64(totalSize))*0.8
					if progress > 0.9 {
						progress = 0.9
					}
					if failed > 0 {
						msg = fmt.Sprintf("Traité: %d / %d MB (New: %d, Reused: %d, ⚠️ Failed: %d chunks)",
							sizeMB, totalSize/(1024*1024), c.newchunk.Load(), c.reusechunk.Load(), failed)
					} else {
						msg = fmt.Sprintf("Traité: %d / %d MB (New: %d, Reused: %d chunks)",
							sizeMB, totalSize/(1024*1024), c.newchunk.Load(), c.reusechunk.Load())
					}
				} else {
					// No total size yet, show indeterminate progress
					progress = 0.1 + float64(sizeMB%100)/1000.0 // Slowly increment from 10%
					if progress > 0.5 {
						progress = 0.5
					}
				}

				// Never report backwards progress - totalSize can increase during backup
				if progress < c.lastProgressPercent {
					progress = c.lastProgressPercent
				}
				c.lastProgressPercent = progress

				c.onProgress(progress, msg)
			}

			c.current_chunk = make([]byte, 0)
			b = b[chunkpos:]
			chunkpos = c.C.Scan(b)
		}
		c.current_chunk = append(c.current_chunk, b...)
	}
	return nil
}

func (c *ChunkState) Eof(client *pbscommon.PBSClient) error {
	if len(c.current_chunk) > 0 {
		h := sha256.New()
		if _, err := h.Write(c.current_chunk); err != nil {
			return fmt.Errorf("failed to hash final chunk: %w", err)
		}

		shahash := hex.EncodeToString(h.Sum(nil))
		if err := binary.Write(c.chunkdigests, binary.LittleEndian, (c.pos + uint64(len(c.current_chunk)))); err != nil {
			return fmt.Errorf("failed to write final chunk offset: %w", err)
		}
		if _, err := c.chunkdigests.Write(h.Sum(nil)); err != nil {
			return fmt.Errorf("failed to write final chunk digest: %w", err)
		}

		if _, ok := c.knownChunks.GetOrInsert(shahash, true); !ok {
			writeBackupLog(fmt.Sprintf("New chunk[%s] %d bytes", shahash, len(c.current_chunk)))

			// Retry final chunk upload with exponential backoff
			chunkData := c.current_chunk
			retryConfig := retry.DefaultConfig()
			retryConfig.MaxAttempts = 5
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			err := retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
				return client.UploadDynamicCompressedChunk(c.wrid, shahash, chunkData)
			})
			if err != nil {
				// Log error but CONTINUE processing
				errMsg := fmt.Sprintf("⚠️  Failed to upload final chunk %s after %d retries: %v", shahash, retryConfig.MaxAttempts, err)
				writeBackupLog(errMsg)
				c.failedchunk.Add(1)

				// Collect error for final report
				c.errorsMutex.Lock()
				c.uploadErrors = append(c.uploadErrors, errMsg)
				c.errorsMutex.Unlock()
			} else {
				c.newchunk.Add(1)
			}
		} else {
			writeBackupLog(fmt.Sprintf("Reuse chunk[%s] %d bytes", shahash, len(c.current_chunk)))
			c.reusechunk.Add(1)
		}
		c.assignments_offset = append(c.assignments_offset, c.pos)
		c.assignments = append(c.assignments, shahash)
		c.pos += uint64(len(c.current_chunk))
		c.chunkcount += 1
	}

	// Assign chunks in batches with retry
	retryConfig := retry.DefaultConfig()
	retryConfig.MaxAttempts = 5
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	for k := 0; k < len(c.assignments); k += 128 {
		k2 := k + 128
		if k2 > len(c.assignments) {
			k2 = len(c.assignments)
		}

		// Capture loop variables for closure
		batchStart, batchEnd := k, k2
		assignments := c.assignments[batchStart:batchEnd]
		offsets := c.assignments_offset[batchStart:batchEnd]

		err := retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
			return client.AssignDynamicChunks(c.wrid, assignments, offsets)
		})
		if err != nil {
			return fmt.Errorf("failed to assign chunks (batch %d-%d) after retries: %w", batchStart, batchEnd, err)
		}
	}

	// Close index with retry
	digest := hex.EncodeToString(c.chunkdigests.Sum(nil))
	err := retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
		return client.CloseDynamicIndex(c.wrid, digest, c.pos, c.chunkcount)
	})
	if err != nil {
		return fmt.Errorf("failed to close dynamic index after retries: %w", err)
	}
	return nil
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// RunBackupInline performs a backup without external binaries
func RunBackupInline(opts BackupOptions) error {
	startTime := time.Now()
	writeBackupLog("Starting inline backup")

	// Validate options
	writeBackupLog("[DEBUG] Validating backup options")
	if opts.BaseURL == "" || opts.AuthID == "" || opts.Secret == "" {
		return fmt.Errorf("PBS connection parameters required")
	}
	writeBackupLog("[DEBUG] Options validated")

	if len(opts.BackupDirs) == 0 {
		return fmt.Errorf("at least one backup directory or drive required")
	}

	// Get hostname for backup-id generation
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unnamed-backup"
	}

	// Auto-split: Analyze directories and split if > 100GB
	// BUT: Check which folders already have backups (skip those with existing snapshots)
	writeBackupLog("[Auto-Split] Analyzing backup directories for automatic splitting...")
	analysis, err := AnalyzeBackupDirs(opts.BackupDirs)
	if err != nil {
		writeBackupLog(fmt.Sprintf("[Auto-Split] Analysis failed: %v - continuing without split", err))
	} else {
		writeBackupLog(fmt.Sprintf("[Auto-Split] Total size: %s, Should split: %v", FormatSize(analysis.TotalSize), analysis.ShouldSplit))

		// Generate base backup-id (used for checking existing backups and creating splits)
		baseBackupID := opts.BackupID
		if baseBackupID == "" {
			baseBackupID = GenerateBackupID(hostname, opts.BackupDirs[0])
		}

		if analysis.ShouldSplit {
			// Check each folder individually for existing backups
			compressionLevel := pbscommon.ParseCompressionLevel(opts.Compression)
			tempClient := &pbscommon.PBSClient{
				BaseURL:          opts.BaseURL,
				CertFingerPrint:  opts.CertFingerprint,
				AuthID:           opts.AuthID,
				Secret:           opts.Secret,
				Datastore:        opts.Datastore,
				Namespace:        opts.Namespace,
				Insecure:         opts.CertFingerprint != "",
				CompressionLevel: compressionLevel,
			}

			allFoldersBackedUp := true
			for i := range analysis.Folders {
				folderBackupID := GenerateBackupID(hostname, analysis.Folders[i].Path)
				analysis.Folders[i].BackupID = folderBackupID

				// Check if this specific folder has a previous backup
				tempClient.Manifest = pbscommon.BackupManifest{BackupID: folderBackupID}
				tempClient.Connect(false, "host")
				previousDidx, err := tempClient.DownloadPreviousToBytes("backup.pxar.didx")

				if err == nil && len(previousDidx) > 0 {
					analysis.Folders[i].BackupExists = true
					writeBackupLog(fmt.Sprintf("[Auto-Split] Folder %s: Previous backup found (skip splitting)", folderBackupID))
				} else {
					analysis.Folders[i].BackupExists = false
					allFoldersBackedUp = false
					writeBackupLog(fmt.Sprintf("[Auto-Split] Folder %s: First backup (will split if large)", folderBackupID))
				}
			}

			// If ALL folders already have backups, disable split entirely (dedup will optimize)
			if allFoldersBackedUp {
				writeBackupLog("[Auto-Split] All folders already backed up - SKIPPING split (deduplication will optimize size)")
				analysis.ShouldSplit = false
			}
		}

		if analysis.ShouldSplit {

			// Create split jobs
			splitJobs := CreateSplitJobs(analysis, baseBackupID, hostname)
			writeBackupLog(fmt.Sprintf("[Auto-Split] Splitting into %d jobs", len(splitJobs)))

			// Execute each split job sequentially
			for _, job := range splitJobs {
				writeBackupLog(fmt.Sprintf("[Auto-Split] Starting job %d/%d: %s (%s, %d folders)",
					job.Index, job.TotalJobs, job.BackupID, FormatSize(job.TotalSize), len(job.Folders)))

				// Create options for this split job
				splitOpts := opts
				splitOpts.BackupDirs = job.Folders
				splitOpts.BackupID = job.BackupID

				// Recursive call for each split (will acquire lock individually)
				if err := runBackupInlineInternal(splitOpts); err != nil {
					// Log error but CONTINUE with remaining jobs
					errMsg := fmt.Sprintf("[Auto-Split] Job %d/%d failed: %v", job.Index, job.TotalJobs, err)
					writeBackupLog(errMsg)
					// Return error so the whole backup is marked as failed
					return fmt.Errorf("%s", errMsg)
				} else {
					writeBackupLog(fmt.Sprintf("[Auto-Split] Job %d/%d completed successfully", job.Index, job.TotalJobs))
				}
			}

			// All split jobs done
			duration := time.Since(startTime)
			writeBackupLog(fmt.Sprintf("[Auto-Split] All %d jobs completed in %s", len(splitJobs), formatDuration(duration)))
			if opts.OnComplete != nil {
				opts.OnComplete(true, fmt.Sprintf("Backup completed (%d split jobs) in %s", len(splitJobs), formatDuration(duration)))
			}
			return nil
		}
	}

	// No split needed, continue with normal backup
	return runBackupInlineInternal(opts)
}

// runBackupInlineInternal is the actual backup implementation (called by RunBackupInline)
func runBackupInlineInternal(opts BackupOptions) error {
	startTime := time.Now()

	// Acquire backup lock for this destination to prevent concurrent backups
	backupLock := getBackupLock(opts.BaseURL, opts.Datastore)
	writeBackupLog(fmt.Sprintf("[Backup Lock] Waiting for lock on %s/%s (prevents concurrent backups)", opts.BaseURL, opts.Datastore))

	// Notify that we're waiting if OnProgress is set
	if opts.OnProgress != nil {
		opts.OnProgress(0, "Waiting for previous backup to complete...")
	}

	backupLock.Lock()
	writeBackupLog(fmt.Sprintf("[Backup Lock] ✓ Lock acquired for %s/%s - starting backup", opts.BaseURL, opts.Datastore))
	defer func() {
		backupLock.Unlock()
		writeBackupLog(fmt.Sprintf("[Backup Lock] ✓ Lock released for %s/%s", opts.BaseURL, opts.Datastore))
	}()

	// Generate backup ID from path if not specified
	writeBackupLog("[DEBUG] Generating backup ID if needed")
	if opts.BackupID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unnamed-backup"
		}

		// Generate backup-id from first directory path: hostname_DRIVE_PATH
		if len(opts.BackupDirs) > 0 {
			opts.BackupID = GenerateBackupID(hostname, opts.BackupDirs[0])
		} else {
			opts.BackupID = hostname
		}
	}
	writeBackupLog(fmt.Sprintf("[DEBUG] BackupID set to: %s", opts.BackupID))

	// Default to "host" type for directory backups
	if opts.BackupType == "" {
		opts.BackupType = "host"
	}

	// Progress callback wrapper
	progress := func(pct float64, msg string) {
		writeBackupLog(fmt.Sprintf("Backup progress: %.1f%% - %s", pct*100, msg))
		if opts.OnProgress != nil {
			opts.OnProgress(pct, msg)
		}
	}

	// Check if all backup directories exist
	writeBackupLog(fmt.Sprintf("[DEBUG] Checking %d backup directories exist", len(opts.BackupDirs)))
	for idx, dir := range opts.BackupDirs {
		writeBackupLog(fmt.Sprintf("[DEBUG] Checking directory %d/%d: %s", idx+1, len(opts.BackupDirs), dir))
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("Backup directory does not exist: %s", dir)
			writeBackupLog(errMsg)
			if opts.OnComplete != nil {
				opts.OnComplete(false, errMsg)
			}
			return fmt.Errorf("%s", errMsg)
		}
	}
	writeBackupLog("[DEBUG] All directories checked, calling progress(0.05)")

	progress(0.05, "Connecting to PBS...")
	writeBackupLog("[DEBUG] After progress(0.05), before connection log")

	// Debug: log connection parameters with sanitized credentials
	writeBackupLog(fmt.Sprintf("PBS Connection: URL=%s, AuthID=%s, Secret=%s, Datastore=%s, BackupID=%s",
		security.SanitizeURL(opts.BaseURL),
		opts.AuthID,
		security.SanitizeSecret(opts.Secret),
		opts.Datastore,
		opts.BackupID))

	writeBackupLog("[DEBUG] Creating PBS client struct")

	// Parse compression level (default to fastest if empty or invalid)
	compressionLevel := pbscommon.ParseCompressionLevel(opts.Compression)
	writeBackupLog(fmt.Sprintf("[DEBUG] Compression level: %s", compressionLevel))

	// Create PBS client
	client := &pbscommon.PBSClient{
		BaseURL:          opts.BaseURL,
		CertFingerPrint:  opts.CertFingerprint,
		AuthID:           opts.AuthID,
		Secret:           opts.Secret,
		Datastore:        opts.Datastore,
		Namespace:        opts.Namespace,
		Insecure:         opts.CertFingerprint != "",
		CompressionLevel: compressionLevel,
		Manifest: pbscommon.BackupManifest{
			BackupID: opts.BackupID,
		},
	}

	writeBackupLog("[DEBUG] PBS client created, starting directory backup loop")

	// Backup each directory
	var newchunk atomic.Uint64
	var reusechunk atomic.Uint64
	var failedchunk atomic.Uint64
	var totalSize atomic.Uint64

	for idx, dir := range opts.BackupDirs {
		// Log directory start but don't call progress() to avoid interfering with ChunkState progress
		writeBackupLog(fmt.Sprintf("Starting backup of directory %d/%d: %s", idx+1, len(opts.BackupDirs), dir))

		err := backupDirectory(client, &newchunk, &reusechunk, &failedchunk, dir, opts.UseVSS, progress)
		if err != nil {
			errMsg := fmt.Sprintf("Backup failed for %s: %v", dir, err)
			writeBackupLog(errMsg)
			if opts.OnComplete != nil {
				opts.OnComplete(false, errMsg)
			}
			return fmt.Errorf("%s", errMsg)
		}
	}

	progress(0.95, "Finalizing backup...")

	// Finalize backup with retry
	retryConfig := retry.DefaultConfig()
	retryConfig.MaxAttempts = 5
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
		return client.Finish()
	})
	if err != nil {
		errMsg := fmt.Sprintf("Failed to finalize backup after retries: %v", err)
		writeBackupLog(errMsg)
		if opts.OnComplete != nil {
			opts.OnComplete(false, errMsg)
		}
		return fmt.Errorf("%s", errMsg)
	}

	progress(1.0, "Backup completed")

	// Calculate backup duration and size
	duration := time.Since(startTime)
	totalSizeMB := float64(totalSize.Load()) / (1024 * 1024)

	// Build completion message with duration, size, and chunk stats
	failed := failedchunk.Load()
	var completionMsg string
	if failed > 0 {
		completionMsg = fmt.Sprintf("⚠️  Backup completed with errors in %s: %.1f MB backed up (%d new, %d reused, %d FAILED chunks)",
			formatDuration(duration), totalSizeMB, newchunk.Load(), reusechunk.Load(), failed)
	} else {
		completionMsg = fmt.Sprintf("Backup completed in %s: %.1f MB backed up (%d new, %d reused chunks)",
			formatDuration(duration), totalSizeMB, newchunk.Load(), reusechunk.Load())
	}

	if len(client.SkippedFiles) > 0 {
		completionMsg += fmt.Sprintf("\n⚠️  %d fichiers/dossiers ignorés (accès refusé ou junction points)", len(client.SkippedFiles))
		writeBackupLog(fmt.Sprintf("=== SKIPPED FILES/DIRECTORIES (%d) ===", len(client.SkippedFiles)))

		// Log first 50 skipped files in detail
		maxLog := 50
		if len(client.SkippedFiles) < maxLog {
			maxLog = len(client.SkippedFiles)
		}
		for i := 0; i < maxLog; i++ {
			writeBackupLog(fmt.Sprintf("  [%d] %s", i+1, client.SkippedFiles[i]))
		}
		if len(client.SkippedFiles) > 50 {
			writeBackupLog(fmt.Sprintf("  ... and %d more (see full list in GUI)", len(client.SkippedFiles)-50))
		}
		writeBackupLog("=== END SKIPPED FILES ===")
	}

	writeBackupLog(fmt.Sprintf("Backup completed: New %d chunks, Reused %d chunks, Skipped %d files",
		newchunk.Load(), reusechunk.Load(), len(client.SkippedFiles)))

	if opts.OnComplete != nil {
		opts.OnComplete(true, completionMsg)
	}

	return nil
}

func backupDirectory(client *pbscommon.PBSClient, newchunk, reusechunk, failedchunk *atomic.Uint64, backupdir string, usevss bool, progress func(float64, string)) error {
	writeBackupLog(fmt.Sprintf("Starting backup of %s", backupdir))

	if usevss {
		return snapshot.CreateVSSSnapshot([]string{backupdir}, func(snaps map[string]snapshot.SnapShot) error {
			for _, snap := range snaps {
				backupdir = snap.FullPath
				break
			}
			return backupReal(client, newchunk, reusechunk, failedchunk, backupdir, progress)
		})
	}

	return backupReal(client, newchunk, reusechunk, failedchunk, backupdir, progress)
}

func backupReal(client *pbscommon.PBSClient, newchunk, reusechunk, failedchunk *atomic.Uint64, backupdir string, progress func(float64, string)) error {
	client.Connect(false, "host")
	knownChunks := hashmap.New[string, bool]()

	// Start keep-alive goroutine to maintain PBS session during long backups
	keepAliveCtx, keepAliveCancel := context.WithCancel(context.Background())
	defer keepAliveCancel() // Stop keep-alive when backup completes or fails

	go func() {
		ticker := time.NewTicker(30 * time.Second) // Keep-alive every 30 seconds
		defer ticker.Stop()

		writeBackupLog("[KeepAlive] Started - will ping PBS every 30 seconds")

		for {
			select {
			case <-keepAliveCtx.Done():
				writeBackupLog("[KeepAlive] Stopped (backup finished)")
				return
			case <-ticker.C:
				writeBackupLog("[KeepAlive] Sending ping to PBS...")
				if err := client.KeepAlive(); err != nil {
					writeBackupLog(fmt.Sprintf("[KeepAlive] WARNING: Ping failed: %v", err))
					// Continue anyway - don't stop the backup for keep-alive failures
				} else {
					writeBackupLog("[KeepAlive] Ping successful")
				}
			}
		}
	}()

	// Start background scan to calculate total size
	totalSize := &atomic.Uint64{}
	go func() {
		writeBackupLog(fmt.Sprintf("Starting background size calculation for: %s", backupdir))
		size, err := calculateDirSize(backupdir)
		if err != nil {
			writeBackupLog(fmt.Sprintf("WARNING: Size calculation had errors: %v", err))
		}
		totalSize.Store(size)
		writeBackupLog(fmt.Sprintf("Total size calculated: %d MB", size/(1024*1024)))
	}()

	archive := &pbscommon.PXARArchive{}
	archive.ArchiveName = "backup.pxar.didx"

	previousDidx, err := client.DownloadPreviousToBytes(archive.ArchiveName)
	if err != nil {
		// This is normal for first backup - no previous backup exists
		writeBackupLog(fmt.Sprintf("No previous backup found (first backup?): %v", err))
		previousDidx = []byte{}
	} else {
		writeBackupLog(fmt.Sprintf("Downloaded previous DIDX: %d bytes", len(previousDidx)))
	}

	if bytes.HasPrefix(previousDidx, didxMagic) {
		previousDidx = previousDidx[4096:]
		for i := 0; i*40 < len(previousDidx); i += 1 {
			e := DidxEntry{}
			e.offset = binary.LittleEndian.Uint64(previousDidx[i*40 : i*40+8])
			e.digest = previousDidx[i*40+8 : i*40+40]
			shahash := hex.EncodeToString(e.digest)
			knownChunks.Set(shahash, true)
		}
	}

	writeBackupLog(fmt.Sprintf("Known chunks: %d", knownChunks.Len()))

	pxarChunk := ChunkState{}
	pxarChunk.Init(newchunk, reusechunk, failedchunk, knownChunks, progress, totalSize)

	pcat1Chunk := ChunkState{}
	pcat1Chunk.Init(newchunk, reusechunk, failedchunk, knownChunks, nil, totalSize)

	pxarChunk.wrid, err = client.CreateDynamicIndex(archive.ArchiveName)
	if err != nil {
		return err
	}
	pcat1Chunk.wrid, err = client.CreateDynamicIndex("catalog.pcat1.didx")
	if err != nil {
		return err
	}

	archive.WriteCB = func(b []byte) error {
		return pxarChunk.HandleData(b, client)
	}

	archive.CatalogWriteCB = func(b []byte) error {
		return pcat1Chunk.HandleData(b, client)
	}

	if _, err = archive.WriteDir(backupdir, "", true); err != nil {
		return fmt.Errorf("failed to write directory archive: %w", err)
	}

	// Collect skipped files from archive
	if len(archive.SkippedFiles) > 0 {
		writeBackupLog(fmt.Sprintf("Backup completed with %d skipped files/directories", len(archive.SkippedFiles)))
		client.SkippedFiles = append(client.SkippedFiles, archive.SkippedFiles...)
	}

	if err = pxarChunk.Eof(client); err != nil {
		return err
	}
	if err = pcat1Chunk.Eof(client); err != nil {
		return err
	}

	// Upload manifest with retry
	retryConfig := retry.DefaultConfig()
	retryConfig.MaxAttempts = 5
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
		return client.UploadManifest()
	})
	if err != nil {
		return fmt.Errorf("failed to upload manifest after retries: %w", err)
	}

	return nil
}
