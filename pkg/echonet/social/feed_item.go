package social

import (
	"github.com/DigiSocialBlock/nexus-protocol/pkg/echonet/core/types"
)

// FeedItem represents a single item in a social feed, combining metadata
// from the NexusContentObjectV1 and the actual resolved content.
type FeedItem struct {
	PostID      string   // From NexusContentObjectV1.ContentId
	AuthorDID   string   // From NexusContentObjectV1.AuthorDid
	Content     string   // The actual post text, retrieved from DDS or direct from ContentBody
	Timestamp   int64    // From NexusContentObjectV1.CreatedAtNetwork or CreatedAtClient
	ContentType types.ContentType // From NexusContentObjectV1.ContentType
	Tags        []string // From NexusContentObjectV1.Tags
	// Add any other fields that might be useful for display in a feed
}

// SortFeedItemsByTimestamp sorts a slice of FeedItems by their Timestamp in descending order (newest first).
// Returns a new slice, does not modify the original.
func SortFeedItemsByTimestamp(items []FeedItem) []FeedItem {
	// Implementation of sorting, if needed here, or can be done by the caller.
	// For now, let's assume the FeedManager will handle sorting after assembling the items.
	// If we add it here, we'd use sort.Slice.
	// Example:
	// sortedItems := make([]FeedItem, len(items))
	// copy(sortedItems, items)
	// sort.Slice(sortedItems, func(i, j int) bool {
	//	return sortedItems[i].Timestamp > sortedItems[j].Timestamp
	// })
	// return sortedItems
	return items // Placeholder, actual sorting will be in FeedManager
}
