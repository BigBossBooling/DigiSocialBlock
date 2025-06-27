package chunking

import (
	"bytes"
	"crypto/sha256"
	"errors"

	manifestpb "github.com/DigiSocialBlock/nexus-protocol/pkg/dds/manifest/types"
	"github.com/btcsuite/btcutil/base58"
	"google.golang.org/protobuf/proto"
)

var (
	ErrInvalidChunkSize = errors.New("chunk size must be positive")
	ErrDataNil          = errors.New("input data cannot be nil")
	ErrManifestNil      = errors.New("manifest cannot be nil for CID generation")
)

const DefaultChunkSize = 256 * 1024 // 256 KiB

// Chunk breaks data into fixed-size chunks.
// The last chunk may be smaller than chunkSize if data size is not a multiple of chunkSize.
func Chunk(data []byte, chunkSize int) ([][]byte, error) {
	if data == nil {
		// Depending on desired behavior, empty data might be valid (resulting in zero chunks)
		// or an error. For now, let's allow it and return zero chunks.
		return [][]byte{}, nil
	}
	if chunkSize <= 0 {
		return nil, ErrInvalidChunkSize
	}

	var chunks [][]byte
	reader := bytes.NewReader(data)
	buffer := make([]byte, chunkSize)

	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" { // string comparison for EOF is brittle, but common for bytes.Reader
				break
			}
			return nil, err // Should be io.EOF, but reader.Read might return other errors
		}
		if n == 0 { // Should not happen if err is nil, but as a safeguard
			break
		}
		// Create a copy of the slice, as buffer is reused
		chunk := make([]byte, n)
		copy(chunk, buffer[:n])
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

// GenerateChunkCID calculates the SHA-256 hash of chunkData and encodes it using Base58BTC.
func GenerateChunkCID(chunkData []byte) (string, error) {
	if chunkData == nil {
		return "", ErrDataNil // Or handle as a specific CID for empty data if needed
	}
	hash := sha256.Sum256(chunkData)
	return base58.Encode(hash[:]), nil
}

// GenerateOriginalContentHash calculates the SHA-256 hash of the original full data.
// Returns the hash as a hex-encoded string.
func GenerateOriginalContentHash(originalData []byte) (string, error) {
	if originalData == nil {
		// Allow hashing of empty data, results in a specific known hash.
		// return "", ErrDataNil
	}
	hash := sha256.Sum256(originalData)
	// Return as hex string for now as per proto comment, can be changed to Base58BTC if preferred
	// For consistency with CIDs, Base58BTC might be better.
	// Let's use Base58BTC for now.
	return base58.Encode(hash[:]), nil
}

// CreateContentManifest constructs a ContentManifestV1 object.
// originalContentHash should be the hash of the *entire* content before chunking.
func CreateContentManifest(chunkCIDs []string, originalContentHash string, originalContentSizeBytes int64, originalMimeType string, originalFilename string, manifestTimestamp int64, customMeta map[string]string) (*manifestpb.ContentManifestV1, error) {
	// Basic validation
	if len(chunkCIDs) == 0 && originalContentSizeBytes > 0 {
		// If there's content, there should be chunks, unless it's truly zero-byte content
		// For zero-byte content, chunkCIDs would be empty, originalContentSizeBytes would be 0.
		// This case implies non-zero content but no chunks, which is an issue.
		// However, zero-byte content *can* have zero chunks.
		// Let's refine this: if size > 0, chunks must exist.
		// If size == 0, chunks can be empty.
		return nil, errors.New("non-empty content must have at least one chunk CID")
	}
	if originalContentHash == "" {
		return nil, errors.New("original content hash cannot be empty")
	}

	return &manifestpb.ContentManifestV1{
		ChunkCids:                   chunkCIDs,
		OriginalContentHash:         originalContentHash,
		OriginalContentSizeBytes:    originalContentSizeBytes,
		OriginalContentMimeType:     originalMimeType,
		OriginalFilename:            originalFilename,
		ManifestCreatedAt:           manifestTimestamp,
		CustomMetadata:              customMeta,
	}, nil
}

// GenerateManifestCID calculates the CID for a ContentManifestV1 object.
// It marshals the manifest to its canonical binary protobuf form,
// then calculates SHA-256 hash and encodes it using Base58BTC.
func GenerateManifestCID(manifest *manifestpb.ContentManifestV1) (string, error) {
	if manifest == nil {
		return "", ErrManifestNil
	}

	// Marshal to canonical binary format
	manifestBytes, err := proto.Marshal(manifest)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(manifestBytes)
	return base58.Encode(hash[:]), nil
}
