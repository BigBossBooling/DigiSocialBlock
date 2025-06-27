package retriever

import (
	"bytes"
	"context"
	"crypto/rand"
	"path/filepath"
	"testing"

	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/chunking"
	manifestpb "github.com/DigiSocialBlock/nexus-protocol/pkg/dds/manifest/types"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/originator"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/service"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func setupTestRetriever(t *testing.T) (*Retriever, *storage.FileSystemStorageManager, *service.StubDDSService) {
	t.Helper()
	basePath := filepath.Join(t.TempDir(), "retriever_storage_test")
	sm, err := storage.NewFileSystemStorageManager(basePath)
	require.NoError(t, err)

	ddsServ := service.NewStubDDSService()
	ret, err := NewRetriever(sm, ddsServ)
	require.NoError(t, err)
	return ret, sm, ddsServ
}

func publishTestContent(t *testing.T, pub *originator.Publisher, data []byte, mime string, filename string) *originator.PublishContentResult {
	t.Helper()
	ctx := context.Background()
	result, err := pub.PublishContent(ctx, data, mime, filename, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	return result
}

func TestNewRetriever(t *testing.T) {
	t.Parallel()
	basePath := filepath.Join(t.TempDir(), "new_retriever_storage")
	sm, err := storage.NewFileSystemStorageManager(basePath)
	require.NoError(t, err)
	ddsServ := service.NewStubDDSService()

	_, err = NewRetriever(nil, ddsServ)
	assert.Error(t, err, "Should fail with nil storage manager")

	_, err = NewRetriever(sm, nil)
	assert.Error(t, err, "Should fail with nil dds service")

	ret, err := NewRetriever(sm, ddsServ)
	assert.NoError(t, err)
	assert.NotNil(t, ret)
}

func TestRetriever_RetrieveContent_LocalSuccess(t *testing.T) {
	t.Parallel()
	retriever, sm, ddsServ := setupTestRetriever(t) // sm is the local storage for retriever
	ctx := context.Background()

	// Use a separate publisher and its storage to prepare content
	pubBasePath := filepath.Join(t.TempDir(), "retriever_pub_storage_local")
	pubSM, err := storage.NewFileSystemStorageManager(pubBasePath)
	require.NoError(t, err)
	publisher, err := originator.NewPublisher(pubSM, ddsServ, chunking.DefaultChunkSize) // Publisher uses its own SM
	require.NoError(t, err)

	originalData := make([]byte, chunking.DefaultChunkSize+50) // 1 full, 1 partial
	_, err = rand.Read(originalData)
	require.NoError(t, err)

	pubResult := publishTestContent(t, publisher, originalData, "text/plain", "local.txt")

	// Manually copy published content (manifest and chunks) to the retriever's storage manager
	// to simulate it being "locally available" to the retriever.
	manifestDataBytes, err := pubSM.Retrieve(pubResult.ManifestCID)
	require.NoError(t, err)
	err = sm.Store(pubResult.ManifestCID, manifestDataBytes)
	require.NoError(t, err)
	for _, chunkCID := range pubResult.ChunkCIDs {
		chunkDataBytes, err := pubSM.Retrieve(chunkCID)
		require.NoError(t, err)
		err = sm.Store(chunkCID, chunkDataBytes)
		require.NoError(t, err)
	}

	// Now retrieve. It should all come from retriever's local storage.
	retrievedData, retrievedManifest, err := retriever.RetrieveContent(ctx, pubResult.ManifestCID)
	require.NoError(t, err)
	require.NotNil(t, retrievedManifest)
	assert.True(t, bytes.Equal(originalData, retrievedData), "Retrieved data should match original")
	assert.True(t, proto.Equal(pubResult.ContentManifest, retrievedManifest), "Retrieved manifest should match original")
}

func TestRetriever_RetrieveContent_NetworkSuccess(t *testing.T) {
	t.Parallel()
	retriever, _, _ := setupTestRetriever(t) // Retriever's local storage starts empty; original ddsServ not used here
	ctx := context.Background()

	// Publisher setup - its storage acts as the "network source" via the StubDDSService
	pubBasePath := filepath.Join(t.TempDir(), "retriever_pub_storage_network")
	networkSM, err := storage.NewFileSystemStorageManager(pubBasePath) // This is the "remote" storage
	require.NoError(t, err)

	// The StubDDSService needs to be aware of the networkSM for its RetrieveChunk mock
	// And the publisher needs to use the same ddsServ to advertise
	// The publisher will use networkSM, and ddsServ will be populated by publisher's AdvertiseProvide
	// and its mockLocalStorage will be populated by publisher storing into networkSM (this is a bit tangled due to stub).

	// A better stub for this test:
	// When ddsServ.RetrieveChunk is called, it retrieves from networkSM.
	// When ddsServ.AdvertiseProvide is called, it populates its mockProviderMap.
	// When publisher.PublishContent is called, it stores into networkSM and calls ddsServ.AdvertiseProvide.

	// Let's refine the StubDDSService for this test:
	// The publisher will use networkSM.
	// The retriever will use its own empty SM.
	// The ddsServ (freshDdsServ instance) will be configured such that:
	//  - AdvertiseProvide (called by publisher) populates its providerMap.
	//  - RetrieveChunk (called by retriever) will use the RetrieveChunkFunc to fetch from networkSM.

	// The retriever will use this freshDdsServ instance.
	freshDdsServ := service.NewStubDDSService()

	// Configure FindProvidersFunc:
	// When the retriever calls FindProviders for a CID that the publisher advertised,
	// this func should return a dummy peer ID. This simulates finding a provider.
	freshDdsServ.FindProvidersFunc = func(ctx context.Context, cid string) ([]service.PeerID, error) {
		// Check if this CID is something the publisher would have stored in networkSM
		// (and thus advertised). We can check networkSM directly here for the test setup.
		if _, err := networkSM.Has(cid); err == nil {
			// If networkSM has it, publisher would have advertised it.
			// Return a dummy peer ID. The actual ID doesn't matter much for the stub
			// as RetrieveChunkFunc below ignores targetPeerID for this specific test.
			t.Logf("[Test DDSService Stub] FindProvidersFunc called for known CID %s, returning mock provider", cid)
			return []service.PeerID{"mock_provider_peer_id"}, nil
		}
		t.Logf("[Test DDSService Stub] FindProvidersFunc called for unknown CID %s, returning no providers", cid)
		return []service.PeerID{}, nil
	}

	// Configure RetrieveChunkFunc:
	// When the retriever calls RetrieveChunk (after FindProviders gives it a peer),
	// this func should simulate fetching the data from networkSM (publisher's storage).
	freshDdsServ.RetrieveChunkFunc = func(ctx context.Context, targetPeerID service.PeerID, cid string) ([]byte, error) {
		t.Logf("[Test DDSService Stub] RetrieveChunkFunc called for CID %s (peer %s), fetching from networkSM", cid, targetPeerID)
		return networkSM.Retrieve(cid)
	}

	retriever.ddsService = freshDdsServ // Assign the configured stub to the retriever

	// The publisher also needs a ddsService to advertise. It uses the same freshDdsServ.
	// The publisher's AdvertiseProvide will populate freshDdsServ.mockProviderMap (internal to stub, used by default FindProviders if FindProvidersFunc is not set).
	// Since we *are* setting FindProvidersFunc, publisher's AdvertiseProvide calls on freshDdsServ
	// will populate a map that our custom FindProvidersFunc above doesn't actually use.
	// This is fine; our custom FindProvidersFunc directly checks networkSM to simulate if a provider *would* exist.
	publisher, err := originator.NewPublisher(networkSM, freshDdsServ, chunking.DefaultChunkSize)
	require.NoError(t, err)

	originalData := make([]byte, chunking.DefaultChunkSize/2) // Small, single chunk
	_, err = rand.Read(originalData)
	require.NoError(t, err)

	pubResult := publishTestContent(t, publisher, originalData, "image/png", "network.png")
	// Content is now in networkSM, and ddsServ's providerMap is populated by AdvertiseProvide.

	// Now retrieve. Retriever's local storage is empty. It should go to network.
	retrievedData, retrievedManifest, err := retriever.RetrieveContent(ctx, pubResult.ManifestCID)
	require.NoError(t, err)
	require.NotNil(t, retrievedManifest)
	assert.True(t, bytes.Equal(originalData, retrievedData), "Retrieved data should match original")
	assert.True(t, proto.Equal(pubResult.ContentManifest, retrievedManifest), "Retrieved manifest should match original")

	// Also verify that the data is now in the retriever's local cache
	hasManifestLocally, _ := retriever.storageMgr.Has(pubResult.ManifestCID)
	assert.True(t, hasManifestLocally, "Manifest should be cached locally after network retrieval")
	for _, chunkCID := range pubResult.ChunkCIDs {
		hasChunkLocally, _ := retriever.storageMgr.Has(chunkCID)
		assert.True(t, hasChunkLocally, "Chunk %s should be cached locally", chunkCID)
	}
}


func TestRetriever_RetrieveContent_ManifestNotFound(t *testing.T) {
	t.Parallel()
	retriever, _, ddsServ := setupTestRetriever(t)
	ctx := context.Background()

	// Configure ddsServ to simulate manifest not found by FindProviders
	ddsServ.FindProvidersFunc = func(ctx context.Context, cid string) ([]service.PeerID, error) {
		if cid == "non_existent_manifest_cid" {
			t.Logf("[Test Stub] FindProvidersFunc called for %s, returning no providers", cid)
			return []service.PeerID{}, nil // No providers
		}
		t.Logf("[Test Stub] FindProvidersFunc called unexpectedly for %s", cid)
		return nil, assert.AnError // Should not be called with other CIDs in this test path
	}

	_, _, err := retriever.RetrieveContent(ctx, "non_existent_manifest_cid")
	assert.Error(t, err)
	// Check that the error message contains the text of ErrManifestRetrievalFailed
	assert.Contains(t, err.Error(), ErrManifestRetrievalFailed.Error(), "Error message should contain manifest retrieval failure text")
	// Check that the error *chain* contains ErrNoProvidersFound
	assert.ErrorIs(t, err, ErrNoProvidersFound, "The error chain should contain ErrNoProvidersFound")
}

func TestRetriever_RetrieveContent_ChunkNotFound(t *testing.T) {
	t.Parallel()
	retriever, sm, ddsServ := setupTestRetriever(t)
	ctx := context.Background()

	// Publisher setup
	pubBasePath := filepath.Join(t.TempDir(), "retriever_pub_storage_chunkfail")
	pubSM, _ := storage.NewFileSystemStorageManager(pubBasePath)
	publisher, _ := originator.NewPublisher(pubSM, ddsServ, chunking.DefaultChunkSize)

	originalData := make([]byte, chunking.DefaultChunkSize+10) // Two chunks
	rand.Read(originalData)
	pubResult := publishTestContent(t, publisher, originalData, "text/plain", "chunkfail.txt")

	// Store manifest in retriever's local storage, but not the chunks
	manifestDataBytes, _ := pubSM.Retrieve(pubResult.ManifestCID)
	sm.Store(pubResult.ManifestCID, manifestDataBytes)

	// Configure ddsServ.RetrieveChunk to fail for the first chunk CID
	firstChunkCID := pubResult.ChunkCIDs[0]
	ddsServ.RetrieveChunkFunc = func(ctx context.Context, targetPeerID service.PeerID, cid string) ([]byte, error) {
		if cid == firstChunkCID {
			return nil, storage.ErrCIDNotFound // Simulate peer not having the chunk
		}
		// For other CIDs (like the second chunk, if it ever got there), retrieve from publisher's storage
		return pubSM.Retrieve(cid)
	}
	// Ensure FindProviders returns something for the chunks, so it attempts RetrieveChunk
	ddsServ.FindProvidersFunc = func(ctx context.Context, cid string) ([]service.PeerID, error) {
		return []service.PeerID{"some_peer"}, nil
	}


	_, _, err := retriever.RetrieveContent(ctx, pubResult.ManifestCID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrChunkRetrievalFailed)
	// The underlying error from RetrieveChunkFunc (storage.ErrCIDNotFound) should also be there.
	// The way errors are wrapped, it might be nested.
	// For now, checking top-level error is sufficient.
}


func TestRetriever_RetrieveContent_ChunkVerificationFail(t *testing.T) {
	t.Parallel()
	retriever, sm, ddsServ := setupTestRetriever(t)
	ctx := context.Background()

	pubBasePath := filepath.Join(t.TempDir(), "retriever_pub_storage_chunkverifyfail")
	pubSM, _ := storage.NewFileSystemStorageManager(pubBasePath)
	publisher, _ := originator.NewPublisher(pubSM, ddsServ, chunking.DefaultChunkSize)

	originalData := []byte("good data for first chunk")
	pubResult := publishTestContent(t, publisher, originalData, "text/plain", "chunkverifyfail.txt")

	// Store manifest locally
	manifestDataBytes, _ := pubSM.Retrieve(pubResult.ManifestCID)
	sm.Store(pubResult.ManifestCID, manifestDataBytes)

	// Configure ddsServ to return corrupted data for the chunk
	targetChunkCID := pubResult.ChunkCIDs[0]
	ddsServ.RetrieveChunkFunc = func(ctx context.Context, targetPeerID service.PeerID, cid string) ([]byte, error) {
		if cid == targetChunkCID {
			return []byte("corrupted data"), nil // Different data means different CID
		}
		return nil, storage.ErrCIDNotFound
	}
	ddsServ.FindProvidersFunc = func(ctx context.Context, cid string) ([]service.PeerID, error) {
		return []service.PeerID{"some_peer"}, nil
	}


	_, _, err := retriever.RetrieveContent(ctx, pubResult.ManifestCID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrChunkVerificationFailed)
}

func TestRetriever_RetrieveContent_FinalHashVerificationFail(t *testing.T) {
	t.Parallel()
	retriever, sm, ddsServ := setupTestRetriever(t)
	ctx := context.Background()

	pubBasePath := filepath.Join(t.TempDir(), "retriever_pub_storage_finalverifyfail")
	pubSM, _ := storage.NewFileSystemStorageManager(pubBasePath)
	publisher, _ := originator.NewPublisher(pubSM, ddsServ, chunking.DefaultChunkSize)

	originalData := []byte("this is the original correct data")
	pubResult := publishTestContent(t, publisher, originalData, "text/plain", "finalverifyfail.txt")

	// Store manifest locally
	manifestDataBytes, _ := pubSM.Retrieve(pubResult.ManifestCID)
	sm.Store(pubResult.ManifestCID, manifestDataBytes)

	// Store one of the chunks locally, but alter its content slightly
	// This assumes only one chunk for simplicity of this test case.
	require.Len(t, pubResult.ChunkCIDs, 1, "This test expects single chunk data")
	chunkCID := pubResult.ChunkCIDs[0]
	originalChunkData, _ := pubSM.Retrieve(chunkCID)
	corruptedChunkData := bytes.Clone(originalChunkData)
	if len(corruptedChunkData) > 0 {
		corruptedChunkData[0] = corruptedChunkData[0] + 1 // Corrupt the first byte
	} else {
		corruptedChunkData = []byte("non-empty but different") // if original was empty
	}

	// Store the corrupted chunk in the retriever's local storage, so it's picked up.
	// IMPORTANT: The CID used for storage must be the *original* chunk's CID from the manifest,
	// even though the data we're storing under that CID is now "corrupted" (won't match this CID).
	// This simulates a scenario where a stored chunk got corrupted *after* its CID was calculated
	// or a malicious peer serves wrong data for a correct CID.
	// The *first* verification (chunkCID vs hash(chunkData)) will FAIL.

	// Let's refine the test:
	// The CHUNK verification should fail first if data is corrupted.
	// To test FINAL hash verification failure, all chunks must pass *their individual* CID checks,
	// but the reassembled content doesn't match the *manifest's original_content_hash*.
	// This implies the manifest itself is wrong, or the set of chunks (while individually valid)
	// don't form the content described by original_content_hash.

	// Scenario for final hash fail:
	// 1. Publish content A (chunks A1, A2), manifest M_A (refs A1, A2, hash_A_orig).
	// 2. Create a new manifest M_B, that references chunks A1, A2, but has original_content_hash of content B.
	// 3. Store M_B and chunks A1, A2 in retriever's local storage.
	// 4. Retrieve using M_B's CID. Chunks A1, A2 will be retrieved and pass their individual CID checks.
	// 5. Reassembly of A1, A2 gives content A. Hashing content A will not match original_content_hash_B in M_B.

	// Simpler: Modify the manifest's OriginalContentHash before storing it locally for the retriever.
	corruptedManifest := proto.Clone(pubResult.ContentManifest).(*manifestpb.ContentManifestV1)
	corruptedManifest.OriginalContentHash = "totallyWrongOriginalHash" // This hash won't match reassembled data

	corruptedManifestBytes, err := proto.Marshal(corruptedManifest)
	require.NoError(t, err)
	sm.Store(pubResult.ManifestCID, corruptedManifestBytes) // Store corrupted manifest under original CID

	// Store original (correct) chunks locally
	for _, cCID := range pubResult.ChunkCIDs {
		chunkData, _ := pubSM.Retrieve(cCID)
		sm.Store(cCID, chunkData)
	}


	_, _, err = retriever.RetrieveContent(ctx, pubResult.ManifestCID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrContentVerificationFailed, "Error was %v", err)
}


func TestRetriever_RetrieveContent_ZeroByteContent(t *testing.T) {
	t.Parallel()
	retriever, sm, ddsServ := setupTestRetriever(t)
	ctx := context.Background()

	pubBasePath := filepath.Join(t.TempDir(), "retriever_pub_zerobyte")
	pubSM, _ := storage.NewFileSystemStorageManager(pubBasePath)
	publisher, _ := originator.NewPublisher(pubSM, ddsServ, chunking.DefaultChunkSize)

	originalData := []byte{} // Zero-byte content
	pubResult := publishTestContent(t, publisher, originalData, "text/plain", "empty.txt")

	// Store manifest locally
	manifestDataBytes, _ := pubSM.Retrieve(pubResult.ManifestCID)
	sm.Store(pubResult.ManifestCID, manifestDataBytes)
	// No chunks to store for zero-byte content

	retrievedData, retrievedManifest, err := retriever.RetrieveContent(ctx, pubResult.ManifestCID)
	require.NoError(t, err)
	require.NotNil(t, retrievedManifest)
	assert.Empty(t, retrievedData, "Retrieved data for zero-byte content should be empty")
	assert.True(t, proto.Equal(pubResult.ContentManifest, retrievedManifest))
	assert.Equal(t, int64(0), retrievedManifest.OriginalContentSizeBytes)
	assert.Empty(t, retrievedManifest.ChunkCids)

	// Test case: Zero-byte content, but manifest's original hash is wrong
	corruptedManifest := proto.Clone(pubResult.ContentManifest).(*manifestpb.ContentManifestV1)
	corruptedManifest.OriginalContentHash = "wrongZeroByteHash"
	corruptedManifestBytes, _ := proto.Marshal(corruptedManifest)

	// Need a new manifest CID for this corrupted manifest if we were to store it under a new name
	// Or, overwrite the existing one for this test path. Let's overwrite.
	err = sm.Store(pubResult.ManifestCID, corruptedManifestBytes)
	require.NoError(t, err)

	_, _, err = retriever.RetrieveContent(ctx, pubResult.ManifestCID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrContentVerificationFailed)

}

// Note: The test `TestRetriever_RetrieveContent_NetworkSuccess` relies on the `publisher`
// populating the `ddsServ.mockProviderMap` (via `AdvertiseProvide`) and `ddsServ.mockLocalStorage`
// (via the `StoreChunk` calls that the publisher makes *to itself* or a simulated peer, which then
// populates `mockLocalStorage` in the current stub design).
// The `Retriever` then uses `ddsServ.FindProviders` (which reads `mockProviderMap`) and
// `ddsServ.RetrieveChunk` (which reads `mockLocalStorage`). This setup works for the stub.
// The `RetrieveChunkFunc` and `FindProvidersFunc` helpers above are for if we wanted even more
// dynamic control from individual tests without relying on the publisher's interaction with the stub.
// The current tests are written to work with the existing StubDDSService structure.
// The `TestRetriever_RetrieveContent_NetworkSuccess` was updated to use a `RetrieveChunkFunc`
// on the `StubDDSService` to make it explicitly fetch from the `networkSM`. This requires
// adding `RetrieveChunkFunc func(...)` to `StubDDSService` and using it in its `RetrieveChunk` method.

// Let's add the func fields to the StubDDSService for cleaner testing
// This means modifying pkg/dds/service/service.go
// For now, I'll proceed without this change and adapt tests if strictly necessary,
// but it's a good pattern. The current network test relies on specific stub behavior.
// The updated TestRetriever_RetrieveContent_NetworkSuccess already uses this pattern.
// The other tests like _ChunkNotFound will also use this.
