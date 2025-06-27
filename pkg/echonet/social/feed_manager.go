package social

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/DigiSocialBlock/nexus-protocol/pkg/dds/retriever"
	"github.com/DigiSocialBlock/nexus-protocol/pkg/echonet/core/types"
	"github.com/btcsuite/btcutil/base58" // For robust CID checking
)

// FeedManager processes a list of content objects and prepares a social feed.
type FeedManager struct {
	contentRetriever *retriever.Retriever
}

// NewFeedManager creates a new FeedManager.
// The contentRetriever is used to fetch content from DDS if it's stored off-chain.
func NewFeedManager(contentRetriever *retriever.Retriever) *FeedManager {
	return &FeedManager{
		contentRetriever: contentRetriever,
	}
}

// isLikelyCID heuristically checks if a string could be a Base58 encoded CID
// as used in this project (SHA256 hash).
func isLikelyCID(s string) bool {
	if len(s) < 40 || len(s) > 50 { // Typical length for Base58 encoded 32-byte hash
		return false
	}
	// Check if all characters are valid Base58 characters.
	// This regex matches if the string consists *only* of Base58 characters.
	// Base58 characters: 1-9, A-H, J-N, P-Z, a-k, m-z
	// (excludes 0, O, I, l to avoid visual ambiguity)
	validBase58Chars := regexp.MustCompile("^[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]+$")
	if !validBase58Chars.MatchString(s) {
		return false
	}

	// More robust: attempt to decode and check length of decoded bytes.
	// A SHA256 hash is 32 bytes.
	decoded, err := base58.Decode(s)
	if err != nil {
		return false // Not valid Base58
	}
	return len(decoded) == 32 // Check if it's a 256-bit hash
}

// GetPublicFeed generates a feed from a given list of NexusContentObjectV1 objects.
// It filters for posts, retrieves content (from DDS if necessary), and sorts them.
func (fm *FeedManager) GetPublicFeed(ctx context.Context, contentObjects []*types.NexusContentObjectV1) ([]FeedItem, error) {
	if fm.contentRetriever == nil {
		// Or return an error, depending on whether direct content-only posts are allowed
		// and if this manager is expected to always handle DDS.
		// For now, let's assume retriever is essential if any content *might* be on DDS.
		// Consider: if contentObjects are guaranteed to have direct content, retriever might be nil.
		// However, the design implies it should be able to fetch.
		// This check should ideally be in NewFeedManager.
		// return nil, errors.New("FeedManager: ContentRetriever is not initialized")
	}

	var feedItems []FeedItem
	var processingErrors []string // To collect errors without failing the whole feed

	for _, obj := range contentObjects {
		if obj == nil {
			continue // Skip nil objects
		}

		// Filter for posts
		if obj.ContentType != types.ContentType_CONTENT_TYPE_POST {
			continue
		}

		var actualContent string
		var err error

		// Determine if ContentBody is direct content or a Manifest CID
		if isLikelyCID(obj.ContentBody) {
			if fm.contentRetriever == nil {
				// Log this issue: cannot retrieve CID-based content without a retriever
				// For now, we'll skip this item or mark content as unavailable
				processingErrors = append(processingErrors, "Skipped CID content due to missing retriever for ContentID: "+obj.ContentId)
				actualContent = "[Content unavailable: Retriever not configured]"
			} else {
				// It's likely a CID, try to retrieve from DDS
				retrievedData, manifest, retrieveErr := fm.contentRetriever.RetrieveContent(ctx, obj.ContentBody)
				if retrieveErr != nil {
					// Log error and potentially skip this item or mark content as unavailable
					processingErrors = append(processingErrors, "Failed to retrieve content for ContentID "+obj.ContentId+": "+retrieveErr.Error())
					actualContent = "[Content unavailable: " + retrieveErr.Error() + "]"
				} else {
					actualContent = string(retrievedData)
					// Potentially use manifest.OriginalMimeType to ensure it's text, etc.
					_ = manifest // Use manifest if needed for other metadata
				}
			}
		} else {
			// Assume it's direct content
			actualContent = obj.ContentBody
		}

		// Use CreatedAtNetwork if available, otherwise CreatedAtClient for timestamp
		timestamp := obj.CreatedAtNetwork
		if timestamp == 0 {
			timestamp = obj.CreatedAtClient
		}

		feedItems = append(feedItems, FeedItem{
			PostID:      obj.ContentId,
			AuthorDID:   obj.AuthorDid,
			Content:     actualContent,
			Timestamp:   timestamp,
			ContentType: obj.ContentType,
			Tags:        obj.Tags, // Assumes Tags is already a string slice
		})
	}

	// Sort feed items by timestamp in descending order (newest first)
	sort.Slice(feedItems, func(i, j int) bool {
		return feedItems[i].Timestamp > feedItems[j].Timestamp
	})

	// If there were processing errors, we might want to return them along with the feed items
	// For now, just returning items. Errors are logged internally or embedded in content.
	// A more robust error handling strategy could be to return a struct { FeedItems []FeedItem; Errors []error }
	if len(processingErrors) > 0 {
		// Log or return these errors appropriately. For this example, we'll just proceed.
		// Consider returning a composite error or a list of errors.
		// fmt.Printf("Encountered errors during feed generation: %s\n", strings.Join(processingErrors, "; "))
	}

	return feedItems, nil
}
