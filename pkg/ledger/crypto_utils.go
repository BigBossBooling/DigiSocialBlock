// pkg/ledger/crypto_utils.go
package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// HashContent generates a SHA256 hash of the given byte slice and returns it as a hex string.
func HashContent(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// GetDeterministicInput returns a deterministically ordered JSON representation of the transaction
// for consistent hashing. It excludes the Signature field for hashing.
func GetDeterministicInput(tx Transaction) []byte {
	// Create a temporary transaction copy without the Signature for hashing.
	// This ensures that the signature itself doesn't affect the hash it signs.
	txForHashing := tx
	txForHashing.Signature = nil // Important: Signature must be excluded from data being hashed for ID

	// Marshal to JSON. Go's json.Marshal sorts map keys by default if the struct uses `json:"key"` tags.
	// For structs, the order of fields is fixed by the struct definition.
	// This makes it deterministic for a given struct.
	jsonData, err := json.Marshal(txForHashing)
	if err != nil {
		// In a real scenario, you'd handle this error more robustly (e.g., return error or panic).
		// For a core utility, panic is acceptable if it indicates a critical programming error
		// that should have been caught during development (e.g., unmarshalable type in payload).
		panic("Failed to marshal transaction for hashing: " + err.Error())
	}
	return jsonData
}

// CalculateMerkleRoot calculates the Merkle root hash for a list of transactions.
// This is crucial for verifying the integrity of transactions within a block.
func CalculateMerkleRoot(transactions []Transaction) string {
	if len(transactions) == 0 {
		return HashContent([]byte{}) // Merkle root of an empty set is typically a hash of empty string/bytes
	}

	var hashes []string
	for _, tx := range transactions {
		// Use the already calculated transaction ID (which is its hash)
		if tx.ID == "" {
			// This should not happen if transactions are created correctly via NewTransaction
			panic("Transaction in CalculateMerkleRoot has empty ID")
		}
		hashes = append(hashes, tx.ID)
	}

	// Sort hashes to ensure deterministic Merkle root calculation.
	// While block includes an ordered list of transactions, the Merkle tree construction
	// itself needs a consistent order of leaf nodes if they are not already ordered
	// by their nature of insertion. Sorting tx.ID strings provides this.
	sort.Strings(hashes)

	// Iteratively combine and hash until only one root hash remains.
	for len(hashes) > 1 {
		// If the number of hashes is odd, duplicate the last hash.
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		var newLevelHashes []string
		for i := 0; i < len(hashes); i += 2 {
			// Concatenate the two hashes. Order matters here (h1+h2 is different from h2+h1).
			// Since we sorted initially, hashes[i] and hashes[i+1] are in a defined order.
			combined := hashes[i] + hashes[i+1]
			newLevelHashes = append(newLevelHashes, HashContent([]byte(combined)))
		}
		hashes = newLevelHashes // Move to the next level of the tree.
	}
	return hashes[0] // The single remaining hash is the Merkle root.
}
