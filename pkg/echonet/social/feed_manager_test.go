package social

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/chunking"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/originator"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/retriever"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/service"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/storage"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/echonet/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// testSetup holds all components needed for a feed manager test environment.
type testSetup struct {
	fm              *FeedManager
	retriever       *retriever.Retriever
	retrieverSM     *storage.FileSystemStorageManager // Retriever's local cache
	ddsServ         *service.StubDDSService         // Mock DDS network service
	publisher       *originator.Publisher
	publisherSM     *storage.FileSystemStorageManager // "Network" storage, where publisher puts content
	ctx             context.Context
	t               *testing.T
}

// newTestSetup creates a new test environment.
func newTestSetup(t *testing.T) *testSetup {
	t.Helper()

	ctx := context.Background()

	// Retriever's storage (local cache)
	retrieverStoragePath := filepath.Join(t.TempDir(), "feedmanager_retriever_storage")
	retrieverSM, err := storage.NewFileSystemStorageManager(retrieverStoragePath)
	require.NoError(t, err)

	// Publisher's storage (simulates network storage)
	publisherStoragePath := filepath.Join(t.TempDir(), "feedmanager_publisher_storage")
	publisherSM, err := storage.NewFileSystemStorageManager(publisherStoragePath)
	require.NoError(t, err)

	// DDS Service (mock)
	ddsServ := service.NewStubDDSService()

	// Retriever
	retr, err := retriever.NewRetriever(retrieverSM, ddsServ)
	require.NoError(t, err)

	// FeedManager
	fm := NewFeedManager(retr)

	// Publisher (to create test content easily)
	// The publisher's ddsServ is also the one the retriever will use.
	// When publisher calls AdvertiseProvide, it populates ddsServ.mockProviderMap.
	// When publisher stores chunks, it uses publisherSM.
	pub, err := originator.NewPublisher(publisherSM, ddsServ, chunking.DefaultChunkSize)
	require.NoError(t, err)

	return &testSetup{
		fm:              fm,
		retriever:       retr,
		retrieverSM:     retrieverSM,
		ddsServ:         ddsServ,
		publisher:       pub,
		publisherSM:     publisherSM,
		ctx:             ctx,
		t:               t,
	}
}

// publishContentForTest is a helper to publish data using the test setup's publisher.
// The content (manifest and chunks) will be stored in publisherSM.
func (ts *testSetup) publishContentForTest(data []byte, mimeType string, filename string) *originator.PublishContentResult {
	ts.t.Helper()
	result, err := ts.publisher.PublishContent(ts.ctx, data, mimeType, filename, nil)
	require.NoError(ts.t, err)
	require.NotNil(ts.t, result)
	return result
}

// Helper to create a NexusContentObjectV1 for testing
func newTestNexusContentObject(id string, author string, contentType types.ContentType, contentBody string, clientTime int64, networkTime int64, tags []string) *types.NexusContentObjectV1 {
	return &types.NexusContentObjectV1{
		ContentId:         id,
		AuthorDid:         author,
		ContentType:       contentType,
		ContentBody:       contentBody,
		CreatedAtClient:   clientTime,
		CreatedAtNetwork:  networkTime,
		Tags:              tags,
	}
}


// --- Test Cases ---

func TestGetPublicFeed_EmptyInput(t *testing.T) {
	ts := newTestSetup(t)

	feedItems, err := ts.fm.GetPublicFeed(ts.ctx, nil)
	assert.NoError(t, err)
	assert.Empty(t, feedItems)

	feedItems, err = ts.fm.GetPublicFeed(ts.ctx, []*types.NexusContentObjectV1{})
	assert.NoError(t, err)
	assert.Empty(t, feedItems)
}

func TestGetPublicFeed_NoPosts(t *testing.T) {
	ts := newTestSetup(t)
	contentObjects := []*types.NexusContentObjectV1{
		newTestNexusContentObject("id1", "author1", types.ContentType_CONTENT_TYPE_ARTICLE, "Article content", time.Now().UnixNano(), time.Now().UnixNano(), nil),
		newTestNexusContentObject("id2", "author2", types.ContentType_CONTENT_TYPE_COMMENT, "Comment content", time.Now().UnixNano(), time.Now().UnixNano(), nil),
	}

	feedItems, err := ts.fm.GetPublicFeed(ts.ctx, contentObjects)
	assert.NoError(t, err)
	assert.Empty(t, feedItems)
}

func TestGetPublicFeed_DirectContentOnly(t *testing.T) {
	ts := newTestSetup(t)
	now := time.Now().UnixNano()
	contentObjects := []*types.NexusContentObjectV1{
		newTestNexusContentObject("post1", "author1", types.ContentType_CONTENT_TYPE_POST, "Direct post 1", now-100, now-100, []string{"tag1"}),
		newTestNexusContentObject("post2", "author2", types.ContentType_CONTENT_TYPE_POST, "Direct post 2", now, now, []string{"tag2"}),
		newTestNexusContentObject("article1", "author1", types.ContentType_CONTENT_TYPE_ARTICLE, "Article", now-50, now-50, nil), // non-post
	}

	feedItems, err := ts.fm.GetPublicFeed(ts.ctx, contentObjects)
	require.NoError(t, err)
	require.Len(t, feedItems, 2)

	// Check order (newest first by CreatedAtNetwork, then CreatedAtClient)
	assert.Equal(t, "post2", feedItems[0].PostID)
	assert.Equal(t, "Direct post 2", feedItems[0].Content)
	assert.Equal(t, now, feedItems[0].Timestamp)
	assert.Equal(t, []string{"tag2"}, feedItems[0].Tags)

	assert.Equal(t, "post1", feedItems[1].PostID)
	assert.Equal(t, "Direct post 1", feedItems[1].Content)
	assert.Equal(t, now-100, feedItems[1].Timestamp)
	assert.Equal(t, []string{"tag1"}, feedItems[1].Tags)
}

func TestGetPublicFeed_DDSContent_RetrieverCacheHit(t *testing.T) {
	ts := newTestSetup(t)
	originalPostData := []byte("This is a post stored on DDS and cached.")
	pubResult := ts.publishContentForTest(originalPostData, "text/plain", "dds_cached.txt")

	// Manually store manifest and chunks in the retriever's cache (ts.retrieverSM)
	manifestBytes, err := ts.publisherSM.Retrieve(pubResult.ManifestCID) // Get from publisher's storage
	require.NoError(t, err)
	err = ts.retrieverSM.Store(pubResult.ManifestCID, manifestBytes) // Store in retriever's cache
	require.NoError(t, err)
	for _, chunkCID := range pubResult.ChunkCIDs {
		chunkBytes, err := ts.publisherSM.Retrieve(chunkCID)
		require.NoError(t, err)
		err = ts.retrieverSM.Store(chunkCID, chunkBytes)
		require.NoError(t, err)
	}

	// Configure DDS service to find no providers, ensuring it's not going to network
	ts.ddsServ.FindProvidersFunc = func(ctx context.Context, cid string) ([]service.PeerID, error) {
		t.Logf("FindProvidersFunc called for %s, returning no providers (forcing cache or fail)", cid)
		return []service.PeerID{}, nil
	}

	now := time.Now().UnixNano()
	contentObjects := []*types.NexusContentObjectV1{
		newTestNexusContentObject("ddsPost1", "authorDDS", types.ContentType_CONTENT_TYPE_POST, pubResult.ManifestCID, now, now, nil),
	}

	feedItems, err := ts.fm.GetPublicFeed(ts.ctx, contentObjects)
	require.NoError(t, err)
	require.Len(t, feedItems, 1)
	assert.Equal(t, "ddsPost1", feedItems[0].PostID)
	assert.Equal(t, string(originalPostData), feedItems[0].Content)
	assert.Equal(t, now, feedItems[0].Timestamp)
}

func TestGetPublicFeed_DDSContent_NetworkSuccess(t *testing.T) {
	ts := newTestSetup(t)
	originalPostData := []byte("This is a post retrieved from the 'network'.")
	pubResult := ts.publishContentForTest(originalPostData, "text/plain", "dds_network.txt")
	// Content is now in ts.publisherSM (the "network") and ts.ddsServ has it advertised.

	// Configure DDS service for successful network retrieval
	ts.ddsServ.FindProvidersFunc = func(ctx context.Context, cid string) ([]service.PeerID, error) {
		// Check if the CID is one that the publisher would have (manifest or chunks)
		// For this test, we assume if publisherSM has it, a provider exists.
		if _, err := ts.publisherSM.Has(cid); err == nil {
			t.Logf("[Test DDSService Stub] FindProvidersFunc called for known CID %s, returning mock provider", cid)
			return []service.PeerID{"mock_network_peer"}, nil
		}
		t.Logf("[Test DDSService Stub] FindProvidersFunc called for unknown CID %s, returning no providers", cid)
		return []service.PeerID{}, nil
	}
	ts.ddsServ.RetrieveChunkFunc = func(ctx context.Context, targetPeerID service.PeerID, cid string) ([]byte, error) {
		t.Logf("[Test DDSService Stub] RetrieveChunkFunc called for CID %s from peer %s, fetching from publisherSM", cid, targetPeerID)
		return ts.publisherSM.Retrieve(cid) // Retrieve from "network" (publisher's storage)
	}

	now := time.Now().UnixNano()
	contentObjects := []*types.NexusContentObjectV1{
		newTestNexusContentObject("ddsNetPost", "authorNet", types.ContentType_CONTENT_TYPE_POST, pubResult.ManifestCID, now, now, nil),
	}

	feedItems, err := ts.fm.GetPublicFeed(ts.ctx, contentObjects)
	require.NoError(t, err)
	require.Len(t, feedItems, 1)
	assert.Equal(t, "ddsNetPost", feedItems[0].PostID)
	assert.Equal(t, string(originalPostData), feedItems[0].Content)

	// Verify content is now cached in retriever's storage
	hasManifest, _ := ts.retrieverSM.Has(pubResult.ManifestCID)
	assert.True(t, hasManifest, "Manifest should be cached by retriever")
	for _, chunkCID := range pubResult.ChunkCIDs {
		hasChunk, _ := ts.retrieverSM.Has(chunkCID)
		assert.True(t, hasChunk, "Chunk %s should be cached", chunkCID)
	}
}


func TestGetPublicFeed_DDSContent_NetworkManifestNotFound(t *testing.T) {
	ts := newTestSetup(t)
	nonExistentCID := "QmNonExistentManifestCID1234567890abcdefghijkl" // A valid looking CID

	ts.ddsServ.FindProvidersFunc = func(ctx context.Context, cid string) ([]service.PeerID, error) {
		if cid == nonExistentCID {
			return []service.PeerID{}, nil // No providers found
		}
		return nil, fmt.Errorf("FindProvidersFunc called unexpectedly for %s", cid)
	}

	now := time.Now().UnixNano()
	contentObjects := []*types.NexusContentObjectV1{
		newTestNexusContentObject("ddsFailPost", "authorFail", types.ContentType_CONTENT_TYPE_POST, nonExistentCID, now, now, nil),
	}

	feedItems, err := ts.fm.GetPublicFeed(ts.ctx, contentObjects)
	require.NoError(t, err) // GetPublicFeed itself shouldn't error, but item content reflects failure
	require.Len(t, feedItems, 1)
	assert.Equal(t, "ddsFailPost", feedItems[0].PostID)
	assert.Contains(t, feedItems[0].Content, "[Content unavailable")
	// Check specific error message if FeedManager was modified to expose individual item errors
}


func TestGetPublicFeed_DDSContent_NetworkChunkNotFound(t *testing.T) {
	ts := newTestSetup(t)
	originalPostData := []byte("Chunk will be missing for this post.")
	pubResult := ts.publishContentForTest(originalPostData, "text/plain", "dds_chunk_missing.txt")
	// Content (manifest M, chunks C1, C2...) is in ts.publisherSM.

	require.True(t, len(pubResult.ChunkCIDs) > 0, "Test requires content with at least one chunk")
	missingChunkCID := pubResult.ChunkCIDs[0]

	ts.ddsServ.FindProvidersFunc = func(ctx context.Context, cid string) ([]service.PeerID, error) {
		// Say provider exists for manifest and all chunks
		return []service.PeerID{"mock_peer_with_some_chunks"}, nil
	}
	ts.ddsServ.RetrieveChunkFunc = func(ctx context.Context, targetPeerID service.PeerID, cid string) ([]byte, error) {
		if cid == missingChunkCID {
			return nil, fmt.Errorf("stub: chunk %s deliberately not found", cid)
		}
		// For manifest and other chunks, retrieve successfully from publisherSM
		return ts.publisherSM.Retrieve(cid)
	}

	now := time.Now().UnixNano()
	contentObjects := []*types.NexusContentObjectV1{
		newTestNexusContentObject("ddsChunkFail", "authorChunkFail", types.ContentType_CONTENT_TYPE_POST, pubResult.ManifestCID, now, now, nil),
	}

	feedItems, err := ts.fm.GetPublicFeed(ts.ctx, contentObjects)
	require.NoError(t, err)
	require.Len(t, feedItems, 1)
	assert.Equal(t, "ddsChunkFail", feedItems[0].PostID)
	assert.Contains(t, feedItems[0].Content, "[Content unavailable")
}

func TestGetPublicFeed_MixedContentAndSorting(t *testing.T) {
	ts := newTestSetup(t)
	now := time.Now().UnixNano()

	// Direct Post (Oldest)
	directPostOld := newTestNexusContentObject("directOld", "authorD", types.ContentType_CONTENT_TYPE_POST, "Old direct post", now-1000, now-1000, nil)

	// DDS Post (Middle)
	ddsPostData := []byte("A post from DDS in the middle.")
	pubResultDDS := ts.publishContentForTest(ddsPostData, "text/plain", "dds_middle.txt")
	// Make DDS post retrievable from network
	ts.ddsServ.FindProvidersFunc = func(ctx context.Context, cid string) ([]service.PeerID, error) {
		if _, err := ts.publisherSM.Has(cid); err == nil { return []service.PeerID{"peer1"}, nil }
		return []service.PeerID{}, nil
	}
	ts.ddsServ.RetrieveChunkFunc = func(ctx context.Context, peer service.PeerID, cid string) ([]byte, error) {
		return ts.publisherSM.Retrieve(cid)
	}
	ddsPostMiddle := newTestNexusContentObject("ddsMiddle", "authorC", types.ContentType_CONTENT_TYPE_POST, pubResultDDS.ManifestCID, now-500, now-500, nil)

	// Direct Post (Newest)
	directPostNew := newTestNexusContentObject("directNew", "authorA", types.ContentType_CONTENT_TYPE_POST, "Newest direct post", now, now, nil)

	// Non-Post
	article := newTestNexusContentObject("article1", "authorB", types.ContentType_CONTENT_TYPE_ARTICLE, "Just an article", now-200, now-200, nil)

	contentObjects := []*types.NexusContentObjectV1{
		directPostOld, // Oldest
		ddsPostMiddle, // Middle
		article,       // Non-post, should be filtered
		directPostNew, // Newest
	}
	// Note: Input is deliberately not sorted by time to test FeedManager's sorting

	feedItems, err := ts.fm.GetPublicFeed(ts.ctx, contentObjects)
	require.NoError(t, err)
	require.Len(t, feedItems, 3, "Should only have 3 posts")

	// Check order (newest first) and content
	assert.Equal(t, "directNew", feedItems[0].PostID)
	assert.Equal(t, "Newest direct post", feedItems[0].Content)

	assert.Equal(t, "ddsMiddle", feedItems[1].PostID)
	assert.Equal(t, string(ddsPostData), feedItems[1].Content)

	assert.Equal(t, "directOld", feedItems[2].PostID)
	assert.Equal(t, "Old direct post", feedItems[2].Content)
}

func TestGetPublicFeed_RetrieverNil_CIDContent(t *testing.T) {
	ts := newTestSetup(t)
	// Create a FeedManager with a nil retriever
	fmNilRetriever := NewFeedManager(nil)

	// Create a valid looking CID, but it won't matter as retriever is nil
	// We can use a real one from publisher for realism if we want
	dummyData := []byte("data for nil retriever test")
	pubResult := ts.publishContentForTest(dummyData, "text/plain", "nil_retriever.txt")

	now := time.Now().UnixNano()
	contentObjects := []*types.NexusContentObjectV1{
		newTestNexusContentObject("cidPostNilRet", "authorNil", types.ContentType_CONTENT_TYPE_POST, pubResult.ManifestCID, now, now, nil),
	}

	feedItems, err := fmNilRetriever.GetPublicFeed(ts.ctx, contentObjects)
	require.NoError(t, err)
	require.Len(t, feedItems, 1)
	assert.Equal(t, "cidPostNilRet", feedItems[0].PostID)
	assert.Contains(t, feedItems[0].Content, "[Content unavailable: Retriever not configured]")
}

func TestIsLikelyCID(t *testing.T) {
	// Valid CIDs (Base58 encoded SHA256 - typical length around 44-46)
	// For this test, we'll generate one using the system's method.
	// We need manifestpb.ContentManifestV1 for GenerateManifestCID.
	dummyPbManifest := &manifestpb.ContentManifestV1{ChunkCids: []string{"dummyChunkCIDForTest"}, OriginalContentHash: "dummyHashForTest"}
	validCID, err := chunking.GenerateManifestCID(dummyPbManifest) // This uses the actual project's generation
	require.NoError(t, err, "Failed to generate sample valid CID for testing isLikelyCID")


	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid looking cid", validCID, true},
		{"too short", "QmTooShort", false},
		{"too long", "QmThisStringIsDefinitelyWayTooLongToBeAValidBase58EncodedSHA256HashIndeedItIs", false},
		{"invalid chars (O,I,l,0)", "QmInvalidCharsOOIIll00", false},
		{"contains non-base58", "QmValidButHas!@#$", false},
		{"empty string", "", false},
		{"just text", "This is a normal sentence.", false},
		{"looks like path", "/path/to/something", false},
		{"another valid looking (random base58, 45 chars)", "Z2gABCDEfghijKLMNOpqrstUVWXYz123456789abcdef", true}, // Assuming 32 byte decode
		{"random base58 but not 32 bytes decode", "Z2gABCDEfghijKLMNOpqrstUVWXYz12345", false}, // Decodes but not to 32 bytes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A bit of a hack for the one "valid" case that isn't actually a 32-byte hash
			// isLikelyCID has a base58.Decode check that expects 32 bytes.
			// The "another valid looking" test case is just random base58 chars.
			// We need to ensure it decodes to 32 bytes for it to be true.
			// Let's create a string that IS a 32-byte hash in base58 for that test case.
			var inputStr string
			if tt.name == "another valid looking (random base58, 45 chars)" {
				randomBytes := make([]byte, 32)
				_, err := rand.Read(randomBytes)
				require.NoError(t, err)
				inputStr = base58.Encode(randomBytes)
			} else {
				inputStr = tt.input
			}


			if isLikelyCID(inputStr) != tt.expected {
				// For debugging, print decode result
				decoded, err := base58.Decode(inputStr)
				t.Logf("Input: '%s', Expected: %v, Got: %v. DecodeLen: %d, DecodeErr: %v", inputStr, tt.expected, !tt.expected, len(decoded), err)
			}
			assert.Equal(t, tt.expected, isLikelyCID(inputStr))
		})
	}
}

// More tests can be added for:
// - Timestamps (CreatedAtNetwork vs CreatedAtClient preference)
// - Error handling within GetPublicFeed (e.g. if context is cancelled during DDS retrieval)
// - Specifics of tags, other metadata.
// - Large number of posts for performance (though that's more benchmarking)

[end of pkg/echonet/social/feed_manager_test.go]
