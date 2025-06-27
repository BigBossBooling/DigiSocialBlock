package chunking

import (
	"crypto/rand"
	"crypto/sha256"
	"testing"
	"time"

	manifestpb "github.com/DigiSocialBlock/nexus-protocol/pkg/dds/manifest/types"
	"github.com/btcsuite/btcutil/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestChunk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		data        []byte
		chunkSize   int
		expectedNum int
		expectedErr error
	}{
		{"nil data", nil, DefaultChunkSize, 0, nil},
		{"empty data", []byte{}, DefaultChunkSize, 0, nil},
		{"data smaller than chunk size", []byte("hello"), DefaultChunkSize, 1, nil},
		{"data equal to chunk size", make([]byte, DefaultChunkSize), DefaultChunkSize, 1, nil},
		{"data larger than chunk size", make([]byte, DefaultChunkSize*2+50), DefaultChunkSize, 3, nil},
		{"data is exact multiple of chunk size", make([]byte, DefaultChunkSize*3), DefaultChunkSize, 3, nil},
		{"invalid chunk size (zero)", []byte("test"), 0, 0, ErrInvalidChunkSize},
		{"invalid chunk size (negative)", []byte("test"), -100, 0, ErrInvalidChunkSize},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chunks, err := Chunk(tt.data, tt.chunkSize)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				assert.Nil(t, chunks)
			} else {
				assert.NoError(t, err)
				assert.Len(t, chunks, tt.expectedNum)

				// Verify content and sizes
				totalLen := 0
				for i, chunk := range chunks {
					totalLen += len(chunk)
					if i < tt.expectedNum-1 {
						assert.Len(t, chunk, tt.chunkSize)
					} else {
						// Last chunk
						if len(tt.data) > 0 {
							expectedLastChunkSize := len(tt.data) % tt.chunkSize
							if expectedLastChunkSize == 0 && len(tt.data) >= tt.chunkSize { // handles exact multiple case
								expectedLastChunkSize = tt.chunkSize
							}
							assert.Len(t, chunk, expectedLastChunkSize)
						} else {
							assert.Empty(t, chunk) // Should not happen if data is empty and expectedNum is 0
						}
					}
				}
				if len(tt.data) > 0 {
					assert.Equal(t, len(tt.data), totalLen, "Total length of chunks should equal original data length")
				}
			}
		})
	}
}

func TestGenerateChunkCID(t *testing.T) {
	t.Parallel()
	data := []byte("This is some test data for CID generation.")
	expectedHash := sha256.Sum256(data)
	expectedCID := base58.Encode(expectedHash[:])

	cid, err := GenerateChunkCID(data)
	require.NoError(t, err)
	assert.Equal(t, expectedCID, cid)

	// Test nil data
	_, err = GenerateChunkCID(nil)
	assert.ErrorIs(t, err, ErrDataNil)

	// Test empty data (should produce a specific CID)
	emptyData := []byte{}
	emptyHash := sha256.Sum256(emptyData)
	expectedEmptyCID := base58.Encode(emptyHash[:])
	emptyCid, err := GenerateChunkCID(emptyData)
	require.NoError(t, err)
	assert.Equal(t, expectedEmptyCID, emptyCid)
	assert.NotEmpty(t, emptyCid)
}

func TestGenerateOriginalContentHash(t *testing.T) {
	t.Parallel()
	data := []byte("This is the original content.")
	expectedHashBytes := sha256.Sum256(data)
	// As per implementation, it's Base58 encoded
	expectedHashStr := base58.Encode(expectedHashBytes[:])

	hashStr, err := GenerateOriginalContentHash(data)
	require.NoError(t, err)
	assert.Equal(t, expectedHashStr, hashStr)

	// Test nil data - should be allowed and produce a specific hash
	nilHashStr, err := GenerateOriginalContentHash(nil)
	require.NoError(t, err)
	emptyHashBytes := sha256.Sum256([]byte{})
	expectedEmptyHashStr := base58.Encode(emptyHashBytes[:])
	assert.Equal(t, expectedEmptyHashStr, nilHashStr)

	// Test empty data
	emptyHashStr, err := GenerateOriginalContentHash([]byte{})
	require.NoError(t, err)
	assert.Equal(t, expectedEmptyHashStr, emptyHashStr)
}

func TestCreateContentManifest(t *testing.T) {
	t.Parallel()
	chunkCIDs := []string{"cid1", "cid2"}
	originalHash := "originalHash123"
	size := int64(512 * 1024)
	mime := "text/plain"
	filename := "test.txt"
	ts := time.Now().UnixNano()
	meta := map[string]string{"key": "value"}


	manifest, err := CreateContentManifest(chunkCIDs, originalHash, size, mime, filename, ts, meta)
	require.NoError(t, err)
	assert.NotNil(t, manifest)
	assert.Equal(t, chunkCIDs, manifest.ChunkCids)
	assert.Equal(t, originalHash, manifest.OriginalContentHash)
	assert.Equal(t, size, manifest.OriginalContentSizeBytes)
	assert.Equal(t, mime, manifest.OriginalContentMimeType)
	assert.Equal(t, filename, manifest.OriginalFilename)
	assert.Equal(t, ts, manifest.ManifestCreatedAt)
	assert.Equal(t, meta, manifest.CustomMetadata)

	_, err = CreateContentManifest(nil, originalHash, size, mime, filename, ts, meta)
	assert.Error(t, err, "Should error if chunkCIDs is nil but size > 0")

	_, err = CreateContentManifest([]string{}, originalHash, size, mime, filename, ts, meta)
	assert.Error(t, err, "Should error if chunkCIDs is empty but size > 0")

	// Valid case for zero-byte content
	manifestZB, errZB := CreateContentManifest([]string{}, originalHash, 0, mime, filename, ts, meta)
	require.NoError(t, errZB)
	assert.NotNil(t, manifestZB)
	assert.Empty(t, manifestZB.ChunkCids)
	assert.Equal(t, int64(0), manifestZB.OriginalContentSizeBytes)


	_, err = CreateContentManifest(chunkCIDs, "", size, mime, filename, ts, meta)
	assert.Error(t, err, "Should error if originalContentHash is empty")
}

func TestGenerateManifestCID(t *testing.T) {
	t.Parallel()
	manifest := &manifestpb.ContentManifestV1{
		ChunkCids:                []string{"cid1", "cid2", "cid3"},
		OriginalContentHash:      "originalTestHash",
		OriginalContentSizeBytes: 12345,
		ManifestCreatedAt:        time.Now().UnixNano(),
	}

	manifestBytes, err := proto.Marshal(manifest)
	require.NoError(t, err)
	expectedHash := sha256.Sum256(manifestBytes)
	expectedCID := base58.Encode(expectedHash[:])

	cid, err := GenerateManifestCID(manifest)
	require.NoError(t, err)
	assert.Equal(t, expectedCID, cid)

	// Test nil manifest
	_, err = GenerateManifestCID(nil)
	assert.ErrorIs(t, err, ErrManifestNil)
}


// Test realistic scenario: chunk data, create CIDs, create manifest, get manifest CID
func TestFullChunkingWorkflow(t *testing.T) {
	t.Parallel()
	originalData := make([]byte, DefaultChunkSize*3+100) // 3 full chunks, 1 partial
	_, err := rand.Read(originalData)
	require.NoError(t, err)

	// 1. Generate original content hash
	originalContentHash, err := GenerateOriginalContentHash(originalData)
	require.NoError(t, err)
	require.NotEmpty(t, originalContentHash)

	// 2. Chunk the data
	chunks, err := Chunk(originalData, DefaultChunkSize)
	require.NoError(t, err)
	require.Len(t, chunks, 4)

	// 3. Generate CIDs for each chunk
	var chunkCIDs []string
	for _, chunkData := range chunks {
		cid, err := GenerateChunkCID(chunkData)
		require.NoError(t, err)
		require.NotEmpty(t, cid)
		chunkCIDs = append(chunkCIDs, cid)
	}
	require.Len(t, chunkCIDs, 4)

	// 4. Create ContentManifest
	now := time.Now().UnixNano()
	manifest, err := CreateContentManifest(
		chunkCIDs,
		originalContentHash,
		int64(len(originalData)),
		"application/octet-stream",
		"random.dat",
		now,
		map[string]string{"test_run": "true"},
	)
	require.NoError(t, err)
	require.NotNil(t, manifest)

	// 5. Generate Manifest CID
	manifestCID, err := GenerateManifestCID(manifest)
	require.NoError(t, err)
	require.NotEmpty(t, manifestCID)

	// Basic sanity checks
	assert.Equal(t, chunkCIDs, manifest.ChunkCids)
	assert.Equal(t, originalContentHash, manifest.OriginalContentHash)
	assert.Equal(t, int64(len(originalData)), manifest.OriginalContentSizeBytes)

	// Ensure manifest CID is different from chunk CIDs and original content hash
	for _, chunkCID := range chunkCIDs {
		assert.NotEqual(t, manifestCID, chunkCID)
	}
	assert.NotEqual(t, manifestCID, originalContentHash)
}
