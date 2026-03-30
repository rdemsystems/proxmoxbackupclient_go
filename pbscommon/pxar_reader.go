package pbscommon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PXARReader reads and extracts PXAR archives
// BETA: Basic file extraction only (no symlinks, ACLs, xattrs)
type PXARReader struct {
	data   []byte
	offset int64
}

// PXARHeader represents a generic PXAR entry header
type PXARHeader struct {
	Type uint64
	Size uint64
}

// PXARExtractedFile represents an extracted file with metadata
type PXARExtractedFile struct {
	Path     string
	Size     uint64
	Mode     os.FileMode
	ModTime  int64
	IsDir    bool
	Data     []byte
	Skipped  bool
	SkipReason string
}

// NewPXARReader creates a new PXAR reader from raw data
func NewPXARReader(data []byte) *PXARReader {
	return &PXARReader{
		data:   data,
		offset: 0,
	}
}

// readHeader reads the next PXAR entry header
func (pr *PXARReader) readHeader() (*PXARHeader, error) {
	if pr.offset+16 > int64(len(pr.data)) {
		return nil, io.EOF
	}

	header := &PXARHeader{}
	buf := bytes.NewReader(pr.data[pr.offset : pr.offset+16])

	if err := binary.Read(buf, binary.LittleEndian, &header.Type); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.Size); err != nil {
		return nil, err
	}

	return header, nil
}

// skip advances the offset by n bytes
func (pr *PXARReader) skip(n int64) {
	pr.offset += n
}

// read reads n bytes from current offset
func (pr *PXARReader) read(n int64) ([]byte, error) {
	if pr.offset+n > int64(len(pr.data)) {
		return nil, io.EOF
	}

	data := pr.data[pr.offset : pr.offset+n]
	pr.offset += n
	return data, nil
}

// ExtractAll extracts entire PXAR archive to destination directory
// BETA: Basic implementation - files and directories only
func (pr *PXARReader) ExtractAll(destDir string) ([]PXARExtractedFile, error) {
	extracted := make([]PXARExtractedFile, 0)
	currentPath := ""
	var currentEntry *PXARFileEntry
	var currentFilename string

	for {
		header, err := pr.readHeader()
		if err == io.EOF {
			break
		}
		if err != nil {
			return extracted, fmt.Errorf("failed to read header at offset %d: %w", pr.offset, err)
		}

		// Size includes the header (16 bytes)
		contentSize := int64(header.Size) - 16

		switch header.Type {
		case PXAR_FILENAME:
			// Read filename (null-terminated string)
			pr.skip(16) // Skip header
			nameData, err := pr.read(contentSize)
			if err != nil {
				return extracted, fmt.Errorf("failed to read filename: %w", err)
			}
			// Remove null terminator
			currentFilename = string(bytes.TrimRight(nameData, "\x00"))

		case PXAR_ENTRY, PXAR_ENTRY_V1:
			// Read file entry metadata
			pr.skip(16) // Skip header
			entryData, err := pr.read(contentSize)
			if err != nil {
				return extracted, fmt.Errorf("failed to read entry: %w", err)
			}

			// Parse entry (simplified - just mode and mtime for now)
			if len(entryData) >= 56 {
				currentEntry = &PXARFileEntry{}
				buf := bytes.NewReader(entryData)
				binary.Read(buf, binary.LittleEndian, &currentEntry.mode)
				binary.Read(buf, binary.LittleEndian, &currentEntry.flags)
				binary.Read(buf, binary.LittleEndian, &currentEntry.uid)
				binary.Read(buf, binary.LittleEndian, &currentEntry.gid)
				binary.Read(buf, binary.LittleEndian, &currentEntry.mtime)
			}

		case PXAR_PAYLOAD:
			// File data
			pr.skip(16) // Skip header
			fileData, err := pr.read(contentSize)
			if err != nil {
				return extracted, fmt.Errorf("failed to read payload: %w", err)
			}

			if currentFilename != "" && currentEntry != nil {
				// Build full path
				fullPath := filepath.Join(destDir, currentPath, currentFilename)

				// Create parent directory
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					extracted = append(extracted, PXARExtractedFile{
						Path:       fullPath,
						Skipped:    true,
						SkipReason: fmt.Sprintf("Cannot create directory: %v", err),
					})
					continue
				}

				// Check if it's a directory
				isDir := (currentEntry.mode & IFDIR) != 0

				if isDir {
					// Create directory
					if err := os.MkdirAll(fullPath, os.FileMode(currentEntry.mode&0777)); err != nil {
						extracted = append(extracted, PXARExtractedFile{
							Path:       fullPath,
							IsDir:      true,
							Skipped:    true,
							SkipReason: fmt.Sprintf("Cannot create dir: %v", err),
						})
					} else {
						currentPath = filepath.Join(currentPath, currentFilename)
						extracted = append(extracted, PXARExtractedFile{
							Path:    fullPath,
							IsDir:   true,
							Mode:    os.FileMode(currentEntry.mode & 0777),
							ModTime: int64(currentEntry.mtime.secs),
						})
					}
				} else {
					// Write file
					if err := os.WriteFile(fullPath, fileData, os.FileMode(currentEntry.mode&0777)); err != nil {
						extracted = append(extracted, PXARExtractedFile{
							Path:       fullPath,
							Size:       uint64(len(fileData)),
							Skipped:    true,
							SkipReason: fmt.Sprintf("Cannot write file: %v", err),
						})
					} else {
						// Set modification time
						if currentEntry.mtime.secs > 0 {
							modTime := time.Unix(int64(currentEntry.mtime.secs), 0)
							// Use same time for access and modification
							_ = os.Chtimes(fullPath, modTime, modTime)
						}

						extracted = append(extracted, PXARExtractedFile{
							Path:    fullPath,
							Size:    uint64(len(fileData)),
							Mode:    os.FileMode(currentEntry.mode & 0777),
							ModTime: int64(currentEntry.mtime.secs),
						})
					}
				}

				// Reset for next file
				currentFilename = ""
				currentEntry = nil
			}

		case PXAR_GOODBYE:
			// End of directory - go back up
			pr.skip(int64(header.Size))
			if strings.Contains(currentPath, string(filepath.Separator)) {
				currentPath = filepath.Dir(currentPath)
			} else {
				currentPath = ""
			}

		case PXAR_SYMLINK, PXAR_DEVICE, PXAR_XATTR, PXAR_ACL_USER, PXAR_FCAPS:
			// Skip unsupported features in BETA
			pr.skip(int64(header.Size))

		default:
			// Unknown type - skip
			pr.skip(int64(header.Size))
		}
	}

	return extracted, nil
}
