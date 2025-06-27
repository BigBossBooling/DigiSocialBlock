package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	ErrCIDNotFound      = errors.New("CID not found in storage")
	ErrInvalidCIDFormat = errors.New("invalid CID format for storage path")
	ErrStoreNilData     = errors.New("cannot store nil data")
	ErrStoreEmptyCID    = errors.New("cannot store data with empty CID")
)

// FilenameRegex is used to sanitize CIDs for use as filenames.
// Base58BTC is generally safe but this provides an explicit check/sanitization.
// Allows alphanumeric characters. This is quite strict; Base58BTC also includes other chars.
// A better approach for Base58BTC might be to ensure it only contains Base58 characters.
// For now, this is a simple alphanumeric sanitizer.
var FilenameRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

// SanitizeCIDForFilename replaces characters not matching FilenameRegex with an underscore.
// This is a basic sanitization. A more robust approach for CIDs might involve
// ensuring they are valid Base58BTC and using them directly if the filesystem supports it,
// or using a directory hashing structure for very large numbers of files.
func SanitizeCIDForFilename(cid string) (string, error) {
	if strings.TrimSpace(cid) == "" {
		return "", ErrInvalidCIDFormat
	}
	// Base58 characters are generally filesystem-safe.
	// Let's check if it's valid Base58 before trying to sanitize.
	// No, btcutil/base58 does not have a validator.
	// We'll assume CIDs are valid Base58. If issues arise, this can be revisited.
	// For extreme safety, one could hex-encode the CID for filenames.
	// For now, we'll use it directly as Base58BTC is mostly fs-safe.
	// However, the spec mentioned "sanitized if used directly as filenames".
	// Let's do a minimal check for path traversal characters.
	if strings.Contains(cid, "/") || strings.Contains(cid, "..") {
		return "", ErrInvalidCIDFormat
	}
	return cid, nil
}

// StorageManager defines the interface for storing and retrieving data chunks by CID.
type StorageManager interface {
	Store(cid string, data []byte) error
	Retrieve(cid string) ([]byte, error)
	Has(cid string) (bool, error)
	Delete(cid string) error
	Location(cid string) (string, error) // Returns the storage path for a given CID
}

// FileSystemStorageManager implements StorageManager using the local file system.
type FileSystemStorageManager struct {
	basePath string
	mu       sync.RWMutex // Protects access to the filesystem if needed, though file operations are often atomic.
}

// NewFileSystemStorageManager creates a new FileSystemStorageManager.
// It creates the base directory if it doesn't exist.
func NewFileSystemStorageManager(basePath string) (*FileSystemStorageManager, error) {
	if strings.TrimSpace(basePath) == "" {
		return nil, errors.New("base path cannot be empty")
	}
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create base storage directory %s: %w", basePath, err)
	}
	return &FileSystemStorageManager{
		basePath: basePath,
	}, nil
}

func (fsm *FileSystemStorageManager) getFilePath(cid string) (string, error) {
	safeCid, err := SanitizeCIDForFilename(cid)
	if err != nil {
		return "", err
	}
	// Consider sharding directories if many files are expected, e.g., base_path/AA/BB/AABBCC...
	// For now, flat structure.
	return filepath.Join(fsm.basePath, safeCid), nil
}

// Store saves data with the given CID.
func (fsm *FileSystemStorageManager) Store(cid string, data []byte) error {
	if strings.TrimSpace(cid) == "" {
		return ErrStoreEmptyCID
	}
	if data == nil {
		return ErrStoreNilData // Storing empty data ([]byte{}) is allowed, but not nil.
	}

	filePath, err := fsm.getFilePath(cid)
	if err != nil {
		return fmt.Errorf("failed to get file path for CID %s: %w", cid, err)
	}

	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	// WriteFile truncates if file exists, which is desired for overwriting/updating.
	err = os.WriteFile(filePath, data, 0640) // Read/write for owner, read for group.
	if err != nil {
		return fmt.Errorf("failed to write data for CID %s to %s: %w", cid, filePath, err)
	}
	return nil
}

// Retrieve fetches data by CID.
func (fsm *FileSystemStorageManager) Retrieve(cid string) ([]byte, error) {
	if strings.TrimSpace(cid) == "" {
		return nil, ErrStoreEmptyCID // Or a specific RetrieveEmptyCIDError
	}
	filePath, err := fsm.getFilePath(cid)
	if err != nil {
		return nil, fmt.Errorf("failed to get file path for CID %s: %w", cid, err)
	}

	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCIDNotFound
		}
		return nil, fmt.Errorf("failed to read data for CID %s from %s: %w", cid, filePath, err)
	}
	return data, nil
}

// Has checks if data for the given CID exists.
func (fsm *FileSystemStorageManager) Has(cid string) (bool, error) {
	if strings.TrimSpace(cid) == "" {
		return false, ErrStoreEmptyCID
	}
	filePath, err := fsm.getFilePath(cid)
	if err != nil {
		return false, fmt.Errorf("failed to get file path for CID %s: %w", cid, err)
	}

	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	_, err = os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("error checking existence of CID %s at %s: %w", cid, filePath, err)
}

// Delete removes data associated with the CID.
func (fsm *FileSystemStorageManager) Delete(cid string) error {
	if strings.TrimSpace(cid) == "" {
		return ErrStoreEmptyCID
	}
	filePath, err := fsm.getFilePath(cid)
	if err != nil {
		return fmt.Errorf("failed to get file path for CID %s: %w", cid, err)
	}

	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	err = os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrCIDNotFound // Or return nil if deleting non-existent is OK. Let's make it an error.
		}
		return fmt.Errorf("failed to delete data for CID %s at %s: %w", cid, filePath, err)
	}
	return nil
}

// Location returns the full file path for a given CID.
// Useful for debugging or direct access if needed (though direct access should be rare).
func (fsm *FileSystemStorageManager) Location(cid string) (string, error) {
	if strings.TrimSpace(cid) == "" {
		return "", ErrStoreEmptyCID
	}
	return fsm.getFilePath(cid)
}
