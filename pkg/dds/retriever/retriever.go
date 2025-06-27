package retriever

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log" // Will replace with structured logging

	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/chunking"
	manifestpb "github.com/DigiSocialBlock/nexus-protocol/pkg/dds/manifest/types"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/service"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/storage"
	"errors"

	"github.com/btcsuite/btcutil/base58"
	"google.golang.org/protobuf/proto"
)

var (
	ErrManifestRetrievalFailed = fmt.Errorf("failed to retrieve manifest")
	ErrChunkRetrievalFailed    = fmt.Errorf("failed to retrieve one or more chunks")
	ErrChunkVerificationFailed = fmt.Errorf("chunk data verification failed (CID mismatch)")
	ErrContentVerificationFailed = fmt.Errorf("final content verification failed (hash mismatch with manifest)")
	ErrNoProvidersFound        = fmt.Errorf("no providers found for a required CID")
)

// Retriever handles the process of fetching content based on a manifest CID.
type Retriever struct {
	storageMgr storage.StorageManager // Local cache/storage
	ddsService service.DDSService     // To find providers and fetch chunks
}

// NewRetriever creates a new Retriever.
func NewRetriever(storageMgr storage.StorageManager, ddsService service.DDSService) (*Retriever, error) {
	if storageMgr == nil {
		return nil, fmt.Errorf("storage manager cannot be nil")
	}
	if ddsService == nil {
		return nil, fmt.Errorf("dds service cannot be nil")
	}
	return &Retriever{
		storageMgr: storageMgr,
		ddsService: ddsService,
	}, nil
}

// RetrieveContent fetches, reassembles, and verifies content given a manifest CID.
func (r *Retriever) RetrieveContent(ctx context.Context, manifestCID string) ([]byte, *manifestpb.ContentManifestV1, error) {
	log.Printf("[Retriever] Starting to retrieve content for ManifestCID: %s\n", manifestCID)

	// 1. Retrieve the ContentManifestV1
	//    First, try local storage (cache).
	//    If not found, try to find providers and retrieve from network.
	manifestData, err := r.fetchData(ctx, manifestCID, "manifest")
	if err != nil {
		log.Printf("[Retriever] Error fetching manifest data for CID %s: %v\n", manifestCID, err)
		// Ensure the error from fetchData (which might wrap ErrNoProvidersFound) is properly wrapped.
		return nil, nil, fmt.Errorf("%s (CID: %s): %w", ErrManifestRetrievalFailed.Error(), manifestCID, err)
	}

	contentManifest := &manifestpb.ContentManifestV1{}
	if err := proto.Unmarshal(manifestData, contentManifest); err != nil {
		log.Printf("[Retriever] Error unmarshalling manifest (CID %s): %v\n", manifestCID, err)
		return nil, nil, fmt.Errorf("failed to unmarshal manifest %s: %w", manifestCID, err)
	}
	log.Printf("[Retriever] Successfully retrieved and parsed manifest (CID %s): %+v\n", manifestCID, contentManifest)

	if len(contentManifest.ChunkCids) == 0 {
		// Handle zero-chunk content (e.g., empty file)
		if contentManifest.OriginalContentSizeBytes == 0 {
			log.Printf("[Retriever] Manifest %s indicates zero-byte content with no chunks. Verifying original hash.\n", manifestCID)
			emptyDataHashBytes := sha256.Sum256([]byte{})
			expectedEmptyHash := base58.Encode(emptyDataHashBytes[:])
			if contentManifest.OriginalContentHash != expectedEmptyHash {
				log.Printf("[Retriever] Mismatch for zero-byte content hash. Expected %s, got from manifest %s\n", expectedEmptyHash, contentManifest.OriginalContentHash)
				return nil, contentManifest, fmt.Errorf("%w: hash mismatch for zero-byte content (manifest %s)", ErrContentVerificationFailed, manifestCID)
			}
			log.Printf("[Retriever] Successfully verified zero-byte content for manifest %s.\n", manifestCID)
			return []byte{}, contentManifest, nil
		}
		// This case (no chunks but size > 0) should ideally be caught by manifest validation if one exists.
		log.Printf("[Retriever] Manifest %s has no chunk CIDs but non-zero size. This is unusual.\n", manifestCID)
		return nil, contentManifest, fmt.Errorf("manifest %s has no chunks but claims non-zero content size", manifestCID)

	}

	// 2. For each chunk_cid in the manifest, retrieve and verify
	var reassembledData bytes.Buffer
	retrievedChunks := 0
	for i, chunkCID := range contentManifest.ChunkCids {
		log.Printf("[Retriever] Retrieving chunk %d/%d with CID: %s\n", i+1, len(contentManifest.ChunkCids), chunkCID)

		chunkData, err := r.fetchData(ctx, chunkCID, "chunk")
		if err != nil {
			log.Printf("[Retriever] Error fetching chunk data for CID %s: %v\n", chunkCID, err)
			return nil, contentManifest, fmt.Errorf("%w: chunk %s - %v", ErrChunkRetrievalFailed, chunkCID, err)
		}

		// Verify the retrieved chunk's hash against its chunk_cid
		// Note: GenerateChunkCID uses Base58BTC of SHA256 hash
		actualChunkCID, err := chunking.GenerateChunkCID(chunkData)
		if err != nil {
			log.Printf("[Retriever] Error generating CID for retrieved chunk %s data: %v\n", chunkCID, err)
			return nil, contentManifest, fmt.Errorf("error generating CID for retrieved chunk %s: %w", chunkCID, err)
		}
		if actualChunkCID != chunkCID {
			log.Printf("[Retriever] Chunk CID mismatch for %s. Expected: %s, Got: %s\n", chunkCID, chunkCID, actualChunkCID)
			return nil, contentManifest, fmt.Errorf("%w: CID %s", ErrChunkVerificationFailed, chunkCID)
		}
		log.Printf("[Retriever] Chunk CID %s verified successfully.\n", chunkCID)

		reassembledData.Write(chunkData)
		retrievedChunks++
	}

	if retrievedChunks != len(contentManifest.ChunkCids) {
		log.Printf("[Retriever] Failed to retrieve all chunks. Expected %d, got %d.\n", len(contentManifest.ChunkCids), retrievedChunks)
		return nil, contentManifest, ErrChunkRetrievalFailed
	}
	finalData := reassembledData.Bytes()
	log.Printf("[Retriever] All %d chunks retrieved and reassembled. Total size: %d bytes.\n", retrievedChunks, len(finalData))

	// 3. Verify the reassembled data's hash against original_content_hash from the manifest
	// Note: GenerateOriginalContentHash uses Base58BTC of SHA256 hash
	actualOriginalContentHash, err := chunking.GenerateOriginalContentHash(finalData)
	if err != nil {
		log.Printf("[Retriever] Error generating hash for final reassembled data: %v\n", err)
		return nil, contentManifest, fmt.Errorf("error generating hash for final data: %w", err)
	}

	if actualOriginalContentHash != contentManifest.OriginalContentHash {
		log.Printf("[Retriever] Final content hash mismatch. Expected (from manifest): %s, Got (from reassembled data): %s\n", contentManifest.OriginalContentHash, actualOriginalContentHash)
		return nil, contentManifest, ErrContentVerificationFailed
	}
	log.Printf("[Retriever] Final content verification successful. Hash: %s\n", actualOriginalContentHash)

	// Optionally, cache the reassembled content if it's small enough or frequently accessed.
	// This would involve another CID for the full content, or use the manifest CID with a different prefix.
	// For now, we just return the data.

	return finalData, contentManifest, nil
}

// fetchData is a helper to retrieve data (chunk or manifest) first from local storage,
// then from the network if not found locally.
func (r *Retriever) fetchData(ctx context.Context, cid string, dataType string) ([]byte, error) {
	log.Printf("[Retriever] fetchData: Attempting to retrieve %s CID %s from local storage.\n", dataType, cid)
	data, err := r.storageMgr.Retrieve(cid)
	if err == nil {
		log.Printf("[Retriever] fetchData: %s CID %s found in local storage.\n", dataType, cid)
		return data, nil
	}
	if !errors.Is(err, storage.ErrCIDNotFound) {
		// Unexpected error from local storage
		log.Printf("[Retriever] fetchData: Unexpected error retrieving %s CID %s from local storage: %v\n", dataType, cid, err)
		return nil, err
	}

	// Not in local storage, try fetching from network
	log.Printf("[Retriever] fetchData: %s CID %s not in local storage. Finding providers...\n", dataType, cid)
	providers, err := r.ddsService.FindProviders(ctx, cid)
	if err != nil {
		log.Printf("[Retriever] fetchData: Error finding providers for %s CID %s: %v\n", dataType, cid, err)
		return nil, err
	}
	if len(providers) == 0 {
		log.Printf("[Retriever] fetchData: No providers found for %s CID %s.\n", dataType, cid)
		return nil, fmt.Errorf("%w: for %s CID %s", ErrNoProvidersFound, dataType, cid)
	}
	log.Printf("[Retriever] fetchData: Found %d providers for %s CID %s: %v. Attempting retrieval...\n", len(providers), dataType, cid, providers)

	// Attempt to retrieve from the first provider found (simple strategy for now)
	// In a real implementation, might try multiple providers, select based on latency, etc.
	for _, peerID := range providers {
		log.Printf("[Retriever] fetchData: Attempting to retrieve %s CID %s from peer %s.\n", dataType, cid, peerID)
		networkData, retrieveErr := r.ddsService.RetrieveChunk(ctx, peerID, cid) // Using RetrieveChunk for both manifest and data chunks
		if retrieveErr == nil {
			log.Printf("[Retriever] fetchData: Successfully retrieved %s CID %s from peer %s. DataLen: %d\n", dataType, cid, peerID, len(networkData))
			// Optionally, store in local cache after successful network retrieval
			if storeErr := r.storageMgr.Store(cid, networkData); storeErr != nil {
				log.Printf("[Retriever] fetchData: Error storing %s CID %s to local cache after network retrieval: %v (continuing with retrieved data)\n", dataType, cid, storeErr)
			} else {
				log.Printf("[Retriever] fetchData: %s CID %s stored in local cache.\n", dataType, cid)
			}
			return networkData, nil
		}
		log.Printf("[Retriever] fetchData: Failed to retrieve %s CID %s from peer %s: %v. Trying next provider if available.\n", dataType, cid, peerID, retrieveErr)
	}

	log.Printf("[Retriever] fetchData: Failed to retrieve %s CID %s from any of the %d providers.\n", dataType, cid, len(providers))
	return nil, fmt.Errorf("failed to retrieve %s CID %s from all %d attempted providers", dataType, cid, len(providers))
}
