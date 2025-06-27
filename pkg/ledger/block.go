// pkg/ledger/block.go
package ledger

import (
	"encoding/json"
	"fmt"
	"time"
)

// NewBlock creates a new block for the blockchain.
func NewBlock(index int64, prevBlockHash string, transactions []Transaction) (Block, error) {
	if index < 0 {
		return Block{}, fmt.Errorf("block index cannot be negative")
	}
	// For the genesis block, prevBlockHash will be empty. For others, it must be non-empty.
	if index > 0 && prevBlockHash == "" {
		return Block{}, fmt.Errorf("previous block hash cannot be empty for non-genesis block")
	}

	// Validate transactions before including them in the block
	// This uses the placeholder tx.IsValid() from model.go which delegates to tx.VerifySignature()
	// In a real system, you might have more stringent checks or a mempool that pre-validates.
	for i, tx := range transactions {
		if !tx.VerifySignature() { // Using VerifySignature directly as IsValid is a simple wrapper
			return Block{}, fmt.Errorf("transaction %d (ID: %s) in proposed block is invalid (signature verification failed)", i, tx.ID)
		}
	}

	block := Block{
		Index:         index,
		Timestamp:     time.Now().UnixNano(),
		Transactions:  transactions,
		PrevBlockHash: prevBlockHash,
	}

	// Calculate Merkle Root
	block.MerkleRoot = CalculateMerkleRoot(transactions)
	if block.MerkleRoot == "" && len(transactions) > 0 { // Merkle root of non-empty tx list shouldn't be empty hash of empty
		return Block{}, fmt.Errorf("failed to calculate Merkle root for non-empty transaction list")
	}


	// Calculate block's own hash (hash of block header/content, including Merkle root)
	// For deterministic hashing of the block, marshal it to JSON without its own Hash field.
	blockForHashing := block
	blockForHashing.Hash = "" // Exclude own hash when calculating it

	jsonData, err := json.Marshal(blockForHashing)
	if err != nil {
		return Block{}, fmt.Errorf("failed to marshal block for hashing: %w", err)
	}
	block.Hash = HashContent(jsonData)
	if block.Hash == "" {
		return Block{}, fmt.Errorf("failed to generate block hash (hash is empty)")
	}

	return block, nil
}

// ValidateSelf validates the block's own hash and Merkle root.
// This is a more comprehensive check than Block.IsValid() in model.go.
func (b *Block) ValidateSelf() bool {
	if b.Hash == "" { // A block must have a hash
		return false
	}

	// Recalculate Merkle Root and compare
	// Ensure transactions are not nil before calculating, though CalculateMerkleRoot handles empty.
	if b.Transactions == nil { // Should not happen if block created via NewBlock
		// If transactions can be nil, CalculateMerkleRoot handles it.
		// This check is more about struct integrity.
		// For now, assume NewBlock ensures Transactions is at least an empty slice.
	}
	calculatedMerkleRoot := CalculateMerkleRoot(b.Transactions)
	if calculatedMerkleRoot != b.MerkleRoot {
		return false
	}

	// Recalculate block hash and compare
	blockForHashing := *b     // Create a copy
	blockForHashing.Hash = "" // Exclude own hash when calculating it

	jsonData, err := json.Marshal(blockForHashing)
	if err != nil {
		return false // Failed to marshal, consider invalid
	}
	calculatedHash := HashContent(jsonData)
	return calculatedHash == b.Hash
}
