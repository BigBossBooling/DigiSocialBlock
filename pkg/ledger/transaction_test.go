// pkg/ledger/transaction_test.go
package ledger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransaction(t *testing.T) {
	sender := "testSenderPublicKey"
	txType := TxTypePostCreated
	payload := []byte("This is a test post payload.")

	tx, err := NewTransaction(sender, txType, payload)
	require.NoError(t, err, "NewTransaction should not return an error for valid inputs")

	assert.NotEmpty(t, tx.ID, "Transaction ID should be set and not empty")
	assert.NotZero(t, tx.Timestamp, "Transaction timestamp should be set")
	assert.Equal(t, sender, tx.SenderPublicKey, "SenderPublicKey should match input")
	assert.Equal(t, txType, tx.Type, "Transaction type should match input")
	assert.Equal(t, payload, tx.Payload, "Payload should match input")
	assert.Empty(t, tx.Signature, "Signature should be empty initially")

	// Test ID calculation consistency (deterministic)
	// Need to ensure GetDeterministicInput handles the zero/nil Signature correctly.
	// NewTransaction already calculates ID. We re-calculate to verify.
	expectedID := HashContent(GetDeterministicInput(Transaction{
		Timestamp:       tx.Timestamp, // Use the same timestamp for comparison
		SenderPublicKey: sender,
		Type:            txType,
		Payload:         payload,
		Signature:       nil, // Signature is nil before signing for ID calculation
	}))
	assert.Equal(t, expectedID, tx.ID, "Transaction ID should be the hash of its deterministic input")

	// Test error cases for NewTransaction
	_, err = NewTransaction("", txType, payload)
	assert.Error(t, err, "NewTransaction should error if senderPublicKey is empty")
	// Potentially add tests for invalid txType if validation is added to NewTransaction
}

func TestTransaction_Sign_Placeholder(t *testing.T) {
	tx, _ := NewTransaction("senderSign", TxTypeCommentAdded, []byte("comment payload for signing"))
	require.Empty(t, tx.Signature, "Signature should be empty before signing")

	// privateKey is a placeholder for this test, as actual crypto isn't used yet.
	var placeholderPrivateKey []byte = []byte("dummyPrivateKey")
	err := tx.Sign(placeholderPrivateKey)
	require.NoError(t, err, "Sign method should not return an error (placeholder)")

	assert.NotEmpty(t, tx.Signature, "Signature should be set after signing")
	expectedSig := []byte("placeholder_signature_for_" + tx.ID)
	assert.Equal(t, expectedSig, tx.Signature, "Placeholder signature should match expected format")

	// Test signing a transaction with no ID (should error)
	emptyIDTx := Transaction{}
	err = emptyIDTx.Sign(placeholderPrivateKey)
	assert.Error(t, err, "Signing a transaction with an empty ID should return an error")

}

func TestTransaction_VerifySignature_Placeholder(t *testing.T) {
	tx, _ := NewTransaction("senderVerify", TxTypeLike, []byte("like payload"))

	// Case 1: No signature
	assert.False(t, tx.VerifySignature(), "VerifySignature should be false if no signature is set")

	// Case 2: Valid placeholder signature
	var placeholderPrivateKey []byte = []byte("dummyPrivateKey")
	err := tx.Sign(placeholderPrivateKey) // Signs with placeholder_signature_for_<ID>
	require.NoError(t, err)
	assert.True(t, tx.VerifySignature(), "VerifySignature should be true for a valid placeholder signature")

	// Case 3: Invalid/tampered placeholder signature
	tx.Signature = []byte("tampered_signature_for_" + tx.ID)
	assert.False(t, tx.VerifySignature(), "VerifySignature should be false for a tampered placeholder signature")

	// Case 4: Signature for a different ID
	tx.Signature = []byte("placeholder_signature_for_DIFFERENT_ID")
	assert.False(t, tx.VerifySignature(), "VerifySignature should be false if signature is for a different ID")

	// Case 5: Transaction with no ID
	noIDTx := Transaction{Signature: []byte("some_sig")}
	assert.False(t, noIDTx.VerifySignature(), "VerifySignature should be false for a transaction with no ID")

	// Case 6: Transaction with ID but empty signature byte slice (different from nil)
	idOnlyTx, _ := NewTransaction("senderX", TxTypePostCreated, []byte("p"))
	idOnlyTx.Signature = []byte{}
	assert.False(t, idOnlyTx.VerifySignature(), "VerifySignature should be false for an empty (but not nil) signature")

}

func TestTransaction_IsValid_Model_vs_VerifySignature(t *testing.T) {
	// This test ensures the model's IsValid() and the transaction's VerifySignature()
	// are consistent for the placeholder logic.

	// Valid placeholder signed transaction
	txValid, _ := NewTransaction("sender", TxTypePostCreated, []byte("payload"))
	txValid.Sign([]byte("dummyKey"))
	assert.True(t, txValid.VerifySignature(), "VerifySignature should be true")
	assert.True(t, txValid.IsValid(), "model.IsValid should also be true (checks Signature != nil)")

	// Transaction not signed (signature is nil)
	txNotSigned, _ := NewTransaction("sender", TxTypePostCreated, []byte("payload"))
	assert.False(t, txNotSigned.VerifySignature(), "VerifySignature should be false")
	assert.False(t, txNotSigned.IsValid(), "model.IsValid should also be false (Signature == nil)")

	// Transaction with tampered signature
	txTampered, _ := NewTransaction("sender", TxTypePostCreated, []byte("payload"))
	txTampered.Sign([]byte("dummyKey"))
	txTampered.Signature = []byte("tampered")
	assert.False(t, txTampered.VerifySignature(), "VerifySignature should be false for tampered")
	// model.IsValid will still be true because Signature is not nil.
	// This highlights that model.IsValid is a very basic check.
	assert.True(t, txTampered.IsValid(), "model.IsValid is true if sig is non-nil, even if verification fails")
}
