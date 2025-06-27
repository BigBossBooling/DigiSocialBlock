package originator

import (
	"context"
	"fmt"
	"log" // Will replace with structured logging
	"time"

	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/chunking"
	manifestpb "github.com/DigiSocialBlock/nexus-protocol/pkg/dds/manifest/types"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/service"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/storage"
	"google.golang.org/protobuf/proto"
)

// Publisher handles the process of taking raw content, chunking it,
// storing it locally, creating a manifest, and (conceptually) advertising it.
type Publisher struct {
	storageMgr storage.StorageManager
	ddsService service.DDSService // For advertising content
	chunkSize  int
}

// NewPublisher creates a new Publisher.
func NewPublisher(storageMgr storage.StorageManager, ddsService service.DDSService, chunkSize int) (*Publisher, error) {
	if storageMgr == nil {
		return nil, fmt.Errorf("storage manager cannot be nil")
	}
	if ddsService == nil {
		return nil, fmt.Errorf("dds service cannot be nil")
	}
	if chunkSize <= 0 {
		chunkSize = chunking.DefaultChunkSize
	}
	return &Publisher{
		storageMgr: storageMgr,
		ddsService: ddsService,
		chunkSize:  chunkSize,
	}, nil
}

// PublishContentResult holds the results of publishing content.
type PublishContentResult struct {
	ManifestCID         string
	ChunkCIDs           []string
	OriginalContentHash string
	ContentManifest     *manifestpb.ContentManifestV1
}

// PublishContent processes raw data, chunks it, stores chunks and manifest locally,
// and (conceptually) advertises their availability.
func (p *Publisher) PublishContent(ctx context.Context, data []byte, mimeType string, filename string, customMeta map[string]string) (*PublishContentResult, error) {
	log.Printf("[Publisher] Starting to publish content. DataLen=%d, MimeType=%s, Filename=%s\n", len(data), mimeType, filename)

	// 1. Generate original content hash
	originalContentHash, err := chunking.GenerateOriginalContentHash(data)
	if err != nil {
		log.Printf("[Publisher] Error generating original content hash: %v\n", err)
		return nil, fmt.Errorf("failed to generate original content hash: %w", err)
	}
	log.Printf("[Publisher] Generated OriginalContentHash: %s\n", originalContentHash)

	// 2. Chunk the data
	chunks, err := chunking.Chunk(data, p.chunkSize)
	if err != nil {
		log.Printf("[Publisher] Error chunking data: %v\n", err)
		return nil, fmt.Errorf("failed to chunk data: %w", err)
	}
	log.Printf("[Publisher] Data chunked into %d pieces.\n", len(chunks))

	// 3. Generate CIDs for each chunk and store them
	var chunkCIDs []string
	for i, chunkData := range chunks {
		chunkCID, err := chunking.GenerateChunkCID(chunkData)
		if err != nil {
			log.Printf("[Publisher] Error generating CID for chunk %d: %v\n", i, err)
			return nil, fmt.Errorf("failed to generate CID for chunk %d: %w", i, err)
		}
		chunkCIDs = append(chunkCIDs, chunkCID)

		log.Printf("[Publisher] Storing chunk %d with CID %s\n", i, chunkCID)
		if err := p.storageMgr.Store(chunkCID, chunkData); err != nil {
			log.Printf("[Publisher] Error storing chunk %d (CID %s): %v\n", i, chunkCID, err)
			return nil, fmt.Errorf("failed to store chunk %d (CID %s): %w", i, chunkCID, err)
		}
		log.Printf("[Publisher] Chunk %d (CID %s) stored locally.\n", i, chunkCID)
	}

	// 4. Create ContentManifest
	manifestTimestamp := time.Now().UnixNano()
	contentManifest, err := chunking.CreateContentManifest(
		chunkCIDs,
		originalContentHash,
		int64(len(data)),
		mimeType,
		filename,
		manifestTimestamp,
		customMeta,
	)
	if err != nil {
		log.Printf("[Publisher] Error creating content manifest: %v\n", err)
		return nil, fmt.Errorf("failed to create content manifest: %w", err)
	}
	log.Printf("[Publisher] ContentManifestV1 created: %+v\n", contentManifest)

	// 5. Generate Manifest CID
	manifestCID, err := chunking.GenerateManifestCID(contentManifest)
	if err != nil {
		log.Printf("[Publisher] Error generating manifest CID: %v\n", err)
		return nil, fmt.Errorf("failed to generate manifest CID: %w", err)
	}
	log.Printf("[Publisher] Generated ManifestCID: %s\n", manifestCID)

	// 6. Store the ContentManifest (as bytes) locally using its manifestCID
	manifestBytes, err := proto.Marshal(contentManifest)
	if err != nil {
		log.Printf("[Publisher] Error marshalling manifest (CID %s) for storage: %v\n", manifestCID, err)
		return nil, fmt.Errorf("failed to marshal manifest for storage: %w", err)
	}
	if err := p.storageMgr.Store(manifestCID, manifestBytes); err != nil {
		log.Printf("[Publisher] Error storing manifest (CID %s): %v\n", manifestCID, err)
		return nil, fmt.Errorf("failed to store manifest (CID %s): %w", manifestCID, err)
	}
	log.Printf("[Publisher] Manifest (CID %s) stored locally.\n", manifestCID)

	// 7. (Conceptual) Advertise CIDs to the network/DHT
	// This uses the DDSService stubs for now.
	log.Printf("[Publisher] Advertising manifest CID %s to the network.\n", manifestCID)
	if err := p.ddsService.AdvertiseProvide(ctx, manifestCID); err != nil {
		// Log error but don't fail the whole publish operation for stub,
		// as local storage is the primary goal of this step.
		log.Printf("[Publisher] Error advertising manifest CID %s: %v (continuing)\n", manifestCID, err)
	}

	for i, chunkCID := range chunkCIDs {
		log.Printf("[Publisher] Advertising chunk %d CID %s to the network.\n", i, chunkCID)
		if err := p.ddsService.AdvertiseProvide(ctx, chunkCID); err != nil {
			log.Printf("[Publisher] Error advertising chunk %d CID %s: %v (continuing)\n", i, chunkCID, err)
		}
	}
	log.Printf("[Publisher] Finished advertising CIDs.\n")

	result := &PublishContentResult{
		ManifestCID:         manifestCID,
		ChunkCIDs:           chunkCIDs,
		OriginalContentHash: originalContentHash,
		ContentManifest:     contentManifest,
	}
	log.Printf("[Publisher] Content published successfully. ManifestCID: %s\n", manifestCID)
	return result, nil
}
