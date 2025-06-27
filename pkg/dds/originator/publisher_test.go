package originator

import (
	"context"
	"crypto/rand"
	"path/filepath"
	"testing"

	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/chunking"
	manifestpb "github.com/DigiSocialBlock/nexus-protocol/pkg/dds/manifest/types"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/service"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func setupTestPublisher(t *testing.T) (*Publisher, *storage.FileSystemStorageManager, *service.StubDDSService) {
	t.Helper()
	basePath := filepath.Join(t.TempDir(), "pub_storage_test")
	sm, err := storage.NewFileSystemStorageManager(basePath)
	require.NoError(t, err)

	ddsServ := service.NewStubDDSService()
	pub, err := NewPublisher(sm, ddsServ, chunking.DefaultChunkSize)
	require.NoError(t, err)
	return pub, sm, ddsServ
}

func TestNewPublisher(t *testing.T) {
	t.Parallel()
	basePath := filepath.Join(t.TempDir(), "new_pub_storage")
	sm, err := storage.NewFileSystemStorageManager(basePath)
	require.NoError(t, err)
	ddsServ := service.NewStubDDSService()

	_, err = NewPublisher(nil, ddsServ, chunking.DefaultChunkSize)
	assert.Error(t, err, "Should fail with nil storage manager")

	_, err = NewPublisher(sm, nil, chunking.DefaultChunkSize)
	assert.Error(t, err, "Should fail with nil dds service")

	pub, err := NewPublisher(sm, ddsServ, chunking.DefaultChunkSize)
	assert.NoError(t, err)
	assert.NotNil(t, pub)
	assert.Equal(t, chunking.DefaultChunkSize, pub.chunkSize)

	pubCustomChunk, err := NewPublisher(sm, ddsServ, 1024)
	assert.NoError(t, err)
	assert.Equal(t, 1024, pubCustomChunk.chunkSize)

	pubInvalidChunk, err := NewPublisher(sm, ddsServ, 0) // Should use default
	assert.NoError(t, err)
	assert.Equal(t, chunking.DefaultChunkSize, pubInvalidChunk.chunkSize)
}

func TestPublisher_PublishContent(t *testing.T) {
	t.Parallel()
	publisher, sm, ddsServ := setupTestPublisher(t)
	ctx := context.Background()

	// Test data
	originalData := make([]byte, chunking.DefaultChunkSize*2+150) // 2 full chunks, 1 partial
	_, err := rand.Read(originalData)
	require.NoError(t, err)
	mimeType := "application/octet-stream"
	filename := "test_file.dat"
	customMeta := map[string]string{"source": "publisher_test"}

	result, err := publisher.PublishContent(ctx, originalData, mimeType, filename, customMeta)
	require.NoError(t, err)
	require.NotNil(t, result)

	// 1. Verify ManifestCID and OriginalContentHash are not empty
	assert.NotEmpty(t, result.ManifestCID)
	assert.NotEmpty(t, result.OriginalContentHash)

	// 2. Verify correct number of ChunkCIDs
	expectedNumChunks := (len(originalData) + chunking.DefaultChunkSize - 1) / chunking.DefaultChunkSize
	assert.Len(t, result.ChunkCIDs, expectedNumChunks)
	for _, chunkCID := range result.ChunkCIDs {
		assert.NotEmpty(t, chunkCID)
	}

	// 3. Verify ContentManifest content
	require.NotNil(t, result.ContentManifest)
	assert.Equal(t, result.ChunkCIDs, result.ContentManifest.ChunkCids)
	assert.Equal(t, result.OriginalContentHash, result.ContentManifest.OriginalContentHash)
	assert.Equal(t, int64(len(originalData)), result.ContentManifest.OriginalContentSizeBytes)
	assert.Equal(t, mimeType, result.ContentManifest.OriginalContentMimeType)
	assert.Equal(t, filename, result.ContentManifest.OriginalFilename)
	assert.Equal(t, customMeta, result.ContentManifest.CustomMetadata)
	assert.True(t, result.ContentManifest.ManifestCreatedAt > 0)

	// 4. Verify manifest is stored locally
	storedManifestData, err := sm.Retrieve(result.ManifestCID)
	require.NoError(t, err)
	retrievedManifest := &manifestpb.ContentManifestV1{}
	err = proto.Unmarshal(storedManifestData, retrievedManifest)
	require.NoError(t, err)
	assert.True(t, proto.Equal(result.ContentManifest, retrievedManifest), "Stored manifest should match the one in result")

	// 5. Verify all chunks are stored locally
	for _, chunkCID := range result.ChunkCIDs {
		hasChunk, err := sm.Has(chunkCID)
		assert.NoError(t, err)
		assert.True(t, hasChunk, "Chunk %s should be stored", chunkCID)
	}

	// 6. Verify CIDs were advertised (check stub's mockProviderMap)
	// StubDDSService's AdvertiseProvide adds "this_node_peer_id_stub"
	thisNodeID := service.PeerID("this_node_peer_id_stub")

	manifestProviders, err := ddsServ.FindProviders(ctx, result.ManifestCID)
	assert.NoError(t, err)
	assert.Contains(t, manifestProviders, thisNodeID, "Manifest CID should be advertised")

	for _, chunkCID := range result.ChunkCIDs {
		chunkProviders, err := ddsServ.FindProviders(ctx, chunkCID)
		assert.NoError(t, err)
		assert.Contains(t, chunkProviders, thisNodeID, "Chunk CID %s should be advertised", chunkCID)
	}

	// Test with empty data
	emptyData := []byte{}
	resultEmpty, errEmpty := publisher.PublishContent(ctx, emptyData, "text/plain", "empty.txt", nil)
	require.NoError(t, errEmpty)
	require.NotNil(t, resultEmpty)
	assert.NotEmpty(t, resultEmpty.ManifestCID)
	assert.NotEmpty(t, resultEmpty.OriginalContentHash) // Hash of empty string is specific
	assert.Empty(t, resultEmpty.ChunkCIDs)              // No chunks for empty data
	assert.Equal(t, int64(0), resultEmpty.ContentManifest.OriginalContentSizeBytes)

	storedEmptyManifestData, err := sm.Retrieve(resultEmpty.ManifestCID)
	require.NoError(t, err)
	retrievedEmptyManifest := &manifestpb.ContentManifestV1{}
	err = proto.Unmarshal(storedEmptyManifestData, retrievedEmptyManifest)
	require.NoError(t, err)
	assert.True(t, proto.Equal(resultEmpty.ContentManifest, retrievedEmptyManifest))

	emptyManifestProviders, err := ddsServ.FindProviders(ctx, resultEmpty.ManifestCID)
	assert.NoError(t, err)
	assert.Contains(t, emptyManifestProviders, thisNodeID, "Empty manifest CID should be advertised")
}

func TestPublisher_PublishContent_StorageFailure(t *testing.T) {
	t.Parallel()
	basePath := filepath.Join(t.TempDir(), "pub_storage_fail")

	// Create a storage manager that will fail on store for a specific CID
	failingSM := &mockFailingStorageManager{
		FileSystemStorageManager: nil, // Will be set up
		failOnStoreCID:           "",  // Will be set by chunker
		t:                        t,
	}
	var err error
	failingSM.FileSystemStorageManager, err = storage.NewFileSystemStorageManager(basePath)
	require.NoError(t, err)


	ddsServ := service.NewStubDDSService()
	publisher, err := NewPublisher(failingSM, ddsServ, chunking.DefaultChunkSize)
	require.NoError(t, err)

	originalData := []byte("some data to publish that will fail storage")

	// Determine what the first chunk's CID would be to make it fail
	chunks, _ := chunking.Chunk(originalData, chunking.DefaultChunkSize)
	firstChunkCID, _ := chunking.GenerateChunkCID(chunks[0])
	failingSM.failOnStoreCID = firstChunkCID


	_, err = publisher.PublishContent(context.Background(), originalData, "text/plain", "fail.txt", nil)
	assert.Error(t, err, "PublishContent should fail if storing a chunk fails")
	assert.Contains(t, err.Error(), "failed to store chunk 0", "Error message should indicate chunk storage failure")
}


// --- Mock Failing Storage Manager ---
type mockFailingStorageManager struct {
	*storage.FileSystemStorageManager
	failOnStoreCID string
	t              *testing.T
}

func (m *mockFailingStorageManager) Store(cid string, data []byte) error {
	if cid == m.failOnStoreCID {
		m.t.Logf("MockFailingStorageManager: Intentionally failing store for CID %s", cid)
		return assert.AnError // Predefined error for testing
	}
	return m.FileSystemStorageManager.Store(cid, data)
}
