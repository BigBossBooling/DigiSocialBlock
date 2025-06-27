// pkg/ledger/transaction.go
package ledger

import (
	"fmt"
	"time"
)

// NewTransaction creates a new transaction.
// senderPublicKey: The public key (or address) of the sender.
// txType: The type of transaction (e.g., PostCreated).
// payload: The serialized data relevant to the transaction.
func NewTransaction(senderPublicKey string, txType TransactionType, payload []byte) (Transaction, error) {
	if senderPublicKey == "" {
		return Transaction{}, fmt.Errorf("sender public key cannot be empty")
	}
	// Basic validation for txType could be added here if desired,
	// e.g., checking against a known set of valid TransactionType values.

	tx := Transaction{
		Timestamp:       time.Now().UnixNano(), // High precision timestamp
		SenderPublicKey: senderPublicKey,
		Type:            txType,
		Payload:         payload,
		// ID and Signature will be set after creation/signing
	}

	// Calculate ID (hash) of the transaction content (excluding signature)
	// GetDeterministicInput ensures that the Signature field (which is nil at this point)
	// is correctly handled for hashing.
	tx.ID = HashContent(GetDeterministicInput(tx))
	if tx.ID == "" {
		// This would indicate a problem in HashContent or GetDeterministicInput,
		// or an empty/problematic transaction structure.
		return Transaction{}, fmt.Errorf("failed to generate transaction ID (hash is empty)")
	}
	return tx, nil
}

// Sign signs the transaction's ID using a private key.
// For now, this is a placeholder. Actual signing will be implemented in Task 1.2.
func (tx *Transaction) Sign(privateKey []byte) error {
	// In Task 1.2, this will use crypto/ecdsa or similar.
	// The privateKey argument is a placeholder for the actual key material.
	if tx.ID == "" {
		return fmt.Errorf("cannot sign transaction with empty ID")
	}
	// For now, simulate a signature.
	tx.Signature = []byte("placeholder_signature_for_" + tx.ID)
	return nil
}

// VerifySignature verifies the transaction's signature against its ID and sender's public key.
// For now, this is a placeholder. Actual verification will be implemented in Task 1.2.
func (tx *Transaction) VerifySignature() bool {
	// In Task 1.2, this will use crypto/ecdsa or similar to verify
	// the signature against tx.ID using tx.SenderPublicKey.
	if tx.ID == "" || len(tx.Signature) == 0 {
		return false // Cannot verify if ID or signature is missing
	}
	// For now, just a dummy check.
	expectedSignature := "placeholder_signature_for_" + tx.ID
	return string(tx.Signature) == expectedSignature
}
