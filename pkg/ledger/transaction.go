// pkg/ledger/transaction.go
package ledger

import (
	"crypto/ecdsa" // For ecdsa.PrivateKey, ecdsa.PublicKey type hints if not fully qualified
	"crypto/rand"  // For signing
	"encoding/hex"
	"fmt"
	"time"

	"digisocialblock/pkg/identity" // Import the new identity package
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

// Sign signs the transaction's ID using the provided private key bytes.
// It reconstructs the private key, signs the transaction ID (which is a hex string of a hash),
// and stores the ASN.1 encoded ECDSA signature.
func (tx *Transaction) Sign(privateKeyBytes []byte) error {
	if tx.ID == "" {
		return fmt.Errorf("cannot sign transaction with empty ID")
	}
	if len(privateKeyBytes) == 0 {
		return fmt.Errorf("private key bytes cannot be empty")
	}

	privKey, err := identity.BytesToPrivateKey(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to reconstruct private key: %w", err)
	}

	// tx.ID is a hex string representing the hash of the transaction content.
	// For signing, we need the raw bytes of that hash.
	hashBytes, err := hex.DecodeString(tx.ID)
	if err != nil {
		return fmt.Errorf("failed to decode transaction ID (hex hash) for signing: %w", err)
	}

	// ecdsa.SignASN1 expects the hash of the message to be signed.
	signatureBytes, err := ecdsa.SignASN1(rand.Reader, privKey, hashBytes)
	if err != nil {
		return fmt.Errorf("failed to sign transaction hash: %w", err)
	}

	tx.Signature = signatureBytes
	return nil
}

// VerifySignature verifies the transaction's signature against its ID (hash)
// and the sender's public key (string address).
func (tx *Transaction) VerifySignature() bool {
	if tx.ID == "" || len(tx.Signature) == 0 || tx.SenderPublicKey == "" {
		return false // Cannot verify if ID, signature, or sender public key is missing
	}

	// tx.SenderPublicKey is an address (hex string of marshaled public key).
	// Convert it back to an ecdsa.PublicKey.
	pubKey, err := identity.HexStringToPublicKey(tx.SenderPublicKey)
	if err != nil {
		// Consider logging this error, as it indicates a problem with data integrity or setup.
		// fmt.Printf("Error converting sender public key string to ECDSA public key: %v\n", err)
		return false
	}

	// tx.ID is a hex string representing the hash of the transaction content.
	// Decode it to raw bytes.
	hashBytes, err := hex.DecodeString(tx.ID)
	if err != nil {
		// Consider logging this error.
		// fmt.Printf("Error decoding transaction ID (hex hash) for verification: %v\n", err)
		return false
	}

	// Verify the ASN.1 encoded signature.
	// ecdsa.VerifyASN1 takes the public key, the original hash, and the signature.
	return ecdsa.VerifyASN1(pubKey, hashBytes, tx.Signature)
}
