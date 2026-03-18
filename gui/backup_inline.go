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
	OnProgress      func(percent float64, message string)
	OnComplete      func(success bool, message string)
}

var didxMagic = []byte{28, 145, 78, 165, 25, 186, 179, 205}

// calculateDirSize scans a directory recursively and returns total size in bytes
func calculateDirSize(path string) uint64 {
	var totalSize uint64

	_ = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			totalSize += uint64(info.Size())
		}
		return nil
	})

	return totalSize
}

type ChunkState struct {
	assignments        []string
	assignments_offset []uint64
	pos                uint64
	wrid               uint64
	chunkcount         uint64
	chunkdigests       hash.Hash
	current_chunk      []byte
	C                  pbscommon.Chunker
	newchunk           *atomic.Uint64
	reusechunk         *atomic.Uint64
	knownChunks        *hashmap.Map[string, bool]
	onProgress         func(float64, string)
	lastProgressReport uint64
	totalSize          *atomic.Uint64 // Total size, updated by background scan
}

type DidxEntry struct {
	offset uint64
	digest []byte
}

func (c *ChunkState) Init(newchunk *atomic.Uint64, reusechunk *atomic.Uint64, knownChunks *hashmap.Map[string, bool], onProgress func(float64, string), totalSize *atomic.Uint64) {
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
	c.knownChunks = knownChunks
	c.onProgress = onProgress
	c.lastProgressReport = 0
	c.totalSize = totalSize
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
				writeDebugLog(fmt.Sprintf("New chunk[%s] %d bytes", shahash, len(c.current_chunk)))
				c.newchunk.Add(1)

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
					return fmt.Errorf("failed to upload chunk %s after retries: %w", shahash, err)
				}
			} else {
				writeDebugLog(fmt.Sprintf("Reuse chunk[%s] %d bytes", shahash, len(c.current_chunk)))
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
				msg := fmt.Sprintf("Traité: %d MB (New: %d, Reused: %d chunks)",
					sizeMB, c.newchunk.Load(), c.reusechunk.Load())

				// Calculate progress based on total size if available
				var progress float64
				totalSize := c.totalSize.Load()
				if totalSize > 0 {
					// Progress from 10% to 90% based on bytes processed
					progress = 0.1 + (float64(c.pos)/float64(totalSize))*0.8
					if progress > 0.9 {
						progress = 0.9
					}
					msg = fmt.Sprintf("Traité: %d / %d MB (New: %d, Reused: %d chunks)",
						sizeMB, totalSize/(1024*1024), c.newchunk.Load(), c.reusechunk.Load())
				} else {
					// No total size yet, show indeterminate progress
					progress = 0.1 + float64(sizeMB%100)/1000.0 // Slowly increment from 10%
					if progress > 0.5 {
						progress = 0.5
					}
				}
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
			writeDebugLog(fmt.Sprintf("New chunk[%s] %d bytes", shahash, len(c.current_chunk)))

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
				return fmt.Errorf("failed to upload final chunk %s after retries: %w", shahash, err)
			}
			c.newchunk.Add(1)
		} else {
			writeDebugLog(fmt.Sprintf("Reuse chunk[%s] %d bytes", shahash, len(c.current_chunk)))
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

// RunBackupInline performs a backup without external binaries
func RunBackupInline(opts BackupOptions) error {
	writeDebugLog("Starting inline backup")

	// Validate options
	if opts.BaseURL == "" || opts.AuthID == "" || opts.Secret == "" {
		return fmt.Errorf("PBS connection parameters required")
	}

	if len(opts.BackupDirs) == 0 {
		return fmt.Errorf("At least one backup directory or drive required")
	}

	// Use hostname as backup ID if not specified
	if opts.BackupID == "" {
		hostname, err := os.Hostname()
		if err == nil {
			opts.BackupID = hostname
		} else {
			opts.BackupID = "unnamed-backup"
		}
	}

	// Default to "host" type for directory backups
	if opts.BackupType == "" {
		opts.BackupType = "host"
	}

	// Progress callback wrapper
	progress := func(pct float64, msg string) {
		writeDebugLog(fmt.Sprintf("Backup progress: %.1f%% - %s", pct*100, msg))
		if opts.OnProgress != nil {
			opts.OnProgress(pct, msg)
		}
	}

	// Check if all backup directories exist
	for _, dir := range opts.BackupDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("Backup directory does not exist: %s", dir)
			writeDebugLog(errMsg)
			if opts.OnComplete != nil {
				opts.OnComplete(false, errMsg)
			}
			return fmt.Errorf("%s", errMsg)
		}
	}

	progress(0.05, "Connecting to PBS...")

	// Debug: log connection parameters with sanitized credentials
	writeDebugLog(fmt.Sprintf("PBS Connection: URL=%s, AuthID=%s, Secret=%s, Datastore=%s, BackupID=%s",
		security.SanitizeURL(opts.BaseURL),
		opts.AuthID,
		security.SanitizeSecret(opts.Secret),
		opts.Datastore,
		opts.BackupID))

	// Create PBS client
	client := &pbscommon.PBSClient{
		BaseURL:         opts.BaseURL,
		CertFingerPrint: opts.CertFingerprint,
		AuthID:          opts.AuthID,
		Secret:          opts.Secret,
		Datastore:       opts.Datastore,
		Namespace:       opts.Namespace,
		Insecure:        opts.CertFingerprint != "",
		Manifest: pbscommon.BackupManifest{
			BackupID: opts.BackupID,
		},
	}

	// Backup each directory
	var newchunk atomic.Uint64
	var reusechunk atomic.Uint64

	for idx, dir := range opts.BackupDirs {
		dirProgress := float64(idx) / float64(len(opts.BackupDirs))
		progress(0.1+dirProgress*0.8, fmt.Sprintf("Backing up %s...", dir))

		err := backupDirectory(client, &newchunk, &reusechunk, dir, opts.UseVSS, progress)
		if err != nil {
			errMsg := fmt.Sprintf("Backup failed for %s: %v", dir, err)
			writeDebugLog(errMsg)
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
		writeDebugLog(errMsg)
		if opts.OnComplete != nil {
			opts.OnComplete(false, errMsg)
		}
		return fmt.Errorf("%s", errMsg)
	}

	progress(1.0, "Backup completed")
	writeDebugLog(fmt.Sprintf("Backup completed: New %d chunks, Reused %d chunks", newchunk.Load(), reusechunk.Load()))

	if opts.OnComplete != nil {
		opts.OnComplete(true, fmt.Sprintf("Backup completed: %d new, %d reused chunks", newchunk.Load(), reusechunk.Load()))
	}

	return nil
}

func backupDirectory(client *pbscommon.PBSClient, newchunk, reusechunk *atomic.Uint64, backupdir string, usevss bool, progress func(float64, string)) error {
	writeDebugLog(fmt.Sprintf("Starting backup of %s", backupdir))

	if usevss {
		return snapshot.CreateVSSSnapshot([]string{backupdir}, func(snaps map[string]snapshot.SnapShot) error {
			for _, snap := range snaps {
				backupdir = snap.FullPath
				break
			}
			return backupReal(client, newchunk, reusechunk, backupdir, progress)
		})
	}

	return backupReal(client, newchunk, reusechunk, backupdir, progress)
}

func backupReal(client *pbscommon.PBSClient, newchunk, reusechunk *atomic.Uint64, backupdir string, progress func(float64, string)) error {
	client.Connect(false, "host")
	knownChunks := hashmap.New[string, bool]()

	// Start background scan to calculate total size
	totalSize := &atomic.Uint64{}
	go func() {
		writeDebugLog(fmt.Sprintf("Starting background size calculation for: %s", backupdir))
		size := calculateDirSize(backupdir)
		totalSize.Store(size)
		writeDebugLog(fmt.Sprintf("Total size calculated: %d MB", size/(1024*1024)))
	}()

	archive := &pbscommon.PXARArchive{}
	archive.ArchiveName = "backup.pxar.didx"

	previousDidx, err := client.DownloadPreviousToBytes(archive.ArchiveName)
	if err != nil {
		// This is normal for first backup - no previous backup exists
		writeDebugLog(fmt.Sprintf("No previous backup found (first backup?): %v", err))
		previousDidx = []byte{}
	} else {
		writeDebugLog(fmt.Sprintf("Downloaded previous DIDX: %d bytes", len(previousDidx)))
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

	writeDebugLog(fmt.Sprintf("Known chunks: %d", knownChunks.Len()))

	pxarChunk := ChunkState{}
	pxarChunk.Init(newchunk, reusechunk, knownChunks, progress, totalSize)

	pcat1Chunk := ChunkState{}
	pcat1Chunk.Init(newchunk, reusechunk, knownChunks, nil, totalSize)

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
