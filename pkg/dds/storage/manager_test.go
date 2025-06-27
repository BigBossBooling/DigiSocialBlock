package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileSystemStorageManager(t *testing.T) {
	t.Parallel()
	basePath := filepath.Join(t.TempDir(), "storage_test_new")

	// Test valid creation
	fsm, err := NewFileSystemStorageManager(basePath)
	require.NoError(t, err)
	require.NotNil(t, fsm)
	assert.DirExists(t, basePath, "Base directory should be created")

	// Test creation with existing directory
	fsm2, err2 := NewFileSystemStorageManager(basePath)
	require.NoError(t, err2)
	require.NotNil(t, fsm2)

	// Test invalid path (empty)
	_, err = NewFileSystemStorageManager("")
	assert.Error(t, err, "Should error with empty base path")

	// Test path that cannot be created (e.g. permission denied if run as non-root and path is /root/test)
	// This is harder to test reliably in a sandboxed environment without specific setup.
	// For now, we assume os.MkdirAll handles most typical non-permission errors.
}

func TestSanitizeCIDForFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		cid       string
		expect    string
		expectErr error
	}{
		{"valid cid", "QmValidCID123", "QmValidCID123", nil},
		{"empty cid", " ", "", ErrInvalidCIDFormat},
		{"cid with slash", "QmInvalid/CID", "", ErrInvalidCIDFormat},
		{"cid with dotdot", "QmInvalid..CID", "", ErrInvalidCIDFormat},
		// Base58BTC characters should largely be safe.
		// This sanitizer is currently very simple and relies on direct use of CID.
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sanitized, err := SanitizeCIDForFilename(tt.cid)
			if tt.expectErr != nil {
				assert.ErrorIs(t, err, tt.expectErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expect, sanitized)
			}
		})
	}
}

func TestFileSystemStorageManager_Operations(t *testing.T) {
	t.Parallel()
	basePath := filepath.Join(t.TempDir(), "storage_ops_test")
	fsm, err := NewFileSystemStorageManager(basePath)
	require.NoError(t, err)
	require.NotNil(t, fsm)

	cid1 := "cidForData1"
	data1 := []byte("This is data 1")
	cid2 := "cidForData2"
	data2 := []byte("This is data 2, slightly longer.")
	emptyData := []byte{}
	cidEmpty := "cidForEmptyData"

	// 1. Test Store
	err = fsm.Store(cid1, data1)
	assert.NoError(t, err, "Store should succeed for valid data")

	err = fsm.Store(cid2, data2)
	assert.NoError(t, err, "Store should succeed for more valid data")

	err = fsm.Store(cidEmpty, emptyData)
	assert.NoError(t, err, "Store should succeed for empty data")


	// Verify file existence and content directly (optional, but good for sanity)
	filePath1, _ := fsm.Location(cid1)
	storedData1, fileErr1 := os.ReadFile(filePath1)
	assert.NoError(t, fileErr1)
	assert.Equal(t, data1, storedData1)

	// Test Store errors
	err = fsm.Store("", data1)
	assert.ErrorIs(t, err, ErrStoreEmptyCID, "Store with empty CID should fail")
	err = fsm.Store("cidNilData", nil)
	assert.ErrorIs(t, err, ErrStoreNilData, "Store with nil data should fail")

	// 2. Test Has
	has1, err := fsm.Has(cid1)
	assert.NoError(t, err)
	assert.True(t, has1, "Has should return true for existing CID")

	hasNonExistent, err := fsm.Has("nonExistentCID")
	assert.NoError(t, err)
	assert.False(t, hasNonExistent, "Has should return false for non-existent CID")

	hasEmptyCid, err := fsm.Has("")
	assert.ErrorIs(t, err, ErrStoreEmptyCID)
	assert.False(t, hasEmptyCid)

	hasEmptyDataCid, err := fsm.Has(cidEmpty)
	assert.NoError(t, err)
	assert.True(t, hasEmptyDataCid, "Has should return true for CID with empty data")


	// 3. Test Retrieve
	retrievedData1, err := fsm.Retrieve(cid1)
	assert.NoError(t, err)
	assert.Equal(t, data1, retrievedData1, "Retrieved data should match stored data")

	retrievedData2, err := fsm.Retrieve(cid2)
	assert.NoError(t, err)
	assert.Equal(t, data2, retrievedData2)

	retrievedEmptyData, err := fsm.Retrieve(cidEmpty)
	assert.NoError(t, err)
	assert.Equal(t, emptyData, retrievedEmptyData, "Retrieved empty data should match stored empty data")


	// Test Retrieve errors
	_, err = fsm.Retrieve("nonExistentCID")
	assert.ErrorIs(t, err, ErrCIDNotFound, "Retrieve for non-existent CID should fail with ErrCIDNotFound")

	_, err = fsm.Retrieve("")
	assert.ErrorIs(t, err, ErrStoreEmptyCID, "Retrieve with empty CID should fail")

	// 4. Test Store (overwrite)
	newData1 := []byte("This is updated data 1")
	err = fsm.Store(cid1, newData1)
	assert.NoError(t, err, "Store (overwrite) should succeed")
	retrievedUpdatedData1, err := fsm.Retrieve(cid1)
	assert.NoError(t, err)
	assert.Equal(t, newData1, retrievedUpdatedData1, "Retrieved updated data should match")

	// 5. Test Delete
	err = fsm.Delete(cid1)
	assert.NoError(t, err, "Delete should succeed for existing CID")
	hasAfterDelete, err := fsm.Has(cid1)
	assert.NoError(t, err)
	assert.False(t, hasAfterDelete, "Has should return false after delete")
	_, err = fsm.Retrieve(cid1)
	assert.ErrorIs(t, err, ErrCIDNotFound, "Retrieve after delete should fail")

	// Test Delete errors
	err = fsm.Delete("nonExistentCIDAgain")
	assert.ErrorIs(t, err, ErrCIDNotFound, "Delete for non-existent CID should fail with ErrCIDNotFound")
	err = fsm.Delete("")
	assert.ErrorIs(t, err, ErrStoreEmptyCID, "Delete with empty CID should fail")

	// 6. Test Location
	loc1, err := fsm.Location(cid2) // cid1 was deleted
	assert.NoError(t, err)
	expectedLoc1 := filepath.Join(basePath, cid2)
	assert.Equal(t, expectedLoc1, loc1)

	_, err = fsm.Location("")
	assert.ErrorIs(t, err, ErrStoreEmptyCID)
}

// Test potential concurrency issues if any (though current implementation is basic)
func TestFileSystemStorageManager_Concurrency(t *testing.T) {
	t.Parallel()
	basePath := filepath.Join(t.TempDir(), "storage_concurrency_test")
	fsm, err := NewFileSystemStorageManager(basePath)
	require.NoError(t, err)

	cid := "concurrentCID"
	data := []byte("concurrent data")

	// Simple concurrency test: multiple Stores, then multiple Retrieves
	numGoroutines := 10
	errs := make(chan error, numGoroutines*2)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			errs <- fsm.Store(cid, data) // Multiple goroutines storing the same CID/data (last one wins or they interleave)
		}()
	}

	// Wait for stores to likely complete (not perfect, but for a basic test)
	for i := 0; i < numGoroutines; i++ {
		err := <-errs
		assert.NoError(t, err)
	}

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := fsm.Retrieve(cid)
			errs <- err
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		err := <-errs
		assert.NoError(t, err)
	}

	// Final check
	retrievedData, err := fsm.Retrieve(cid)
	assert.NoError(t, err)
	assert.Equal(t, data, retrievedData)
}
