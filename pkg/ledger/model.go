// pkg/ledger/model.go
package ledger

import (
	"time"
)

// TransactionType defines the type of social action a transaction represents.
type TransactionType string

const (
	TxTypePostCreated    TransactionType = "PostCreated"
	TxTypeCommentAdded   TransactionType = "CommentAdded"
	TxTypeProfileUpdated TransactionType = "ProfileUpdated"
	TxTypeLike           TransactionType = "Like"
	// Add other transaction types as needed for future tasks
)

// Transaction represents a single social action recorded on the ledger.
type Transaction struct {
	// ID is the unique identifier for the transaction, typically the hash of its content.
	ID string `json:"id"`
	// Timestamp when the transaction was created.
	Timestamp int64 `json:"timestamp"`
	// SenderPublicKey is the public key (or address derived from it) of the user who initiated the transaction.
	SenderPublicKey string `json:"sender_public_key"`
	// Type indicates the specific social action (e.g., "PostCreated", "CommentAdded").
	Type TransactionType `json:"type"`
	// Payload holds the serialized data related to the transaction.
	// For example, this could be the serialized NexusContentObjectV1, NexusUserObjectV1, etc.
	Payload []byte `json:"payload"`
	// Signature is the cryptographic signature of the transaction's ID by the SenderPublicKey.
	Signature []byte `json:"signature"`
}

// Block represents a single block in the blockchain.
type Block struct {
	// Index is the block number in the chain.
	Index int64 `json:"index"`
	// Timestamp when the block was created.
	Timestamp int64 `json:"timestamp"`
	// Transactions are the list of transactions included in this block.
	Transactions []Transaction `json:"transactions"`
	// PrevBlockHash is the hash of the previous block in the chain.
	PrevBlockHash string `json:"prev_block_hash"`
	// MerkleRoot is the Merkle tree root hash of all transactions in this block.
	MerkleRoot string `json:"merkle_root"`
	// Hash is the block's own cryptographic hash, unique to its content.
	Hash string `json:"hash"`
	// Nonce (optional, for Proof-of-Work, will be added in a later task if PoW is implemented)
	// Nonce int64 `json:"nonce"`
}

// IsValid checks if the block's own hash is correctly calculated based on its content.
// This is a placeholder for more comprehensive block validation.
// The actual hashing logic will be in crypto_utils and block.ValidateSelf().
func (b *Block) IsValid() bool {
	// This method is a bit redundant given block.ValidateSelf().
	// For now, it can be a simple check or delegate.
	// Let's assume it's a quick check that the Hash field is not empty,
	// with deeper validation in ValidateSelf().
	return b.Hash != "" // Placeholder, actual logic in ValidateSelf
}

// IsValid checks if the transaction's signature is valid.
// This is a placeholder for actual cryptographic verification.
// Actual logic will be in tx.VerifySignature().
func (t *Transaction) IsValid() bool {
	// This method is a bit redundant given tx.VerifySignature().
	// It can be a quick check or delegate.
	return t.Signature != nil // Placeholder, actual logic in VerifySignature
}
