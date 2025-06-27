// pkg/ledger/crypto_utils_test.go
package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashContent(t *testing.T) {
	data1 := []byte("hello world")
	hash1 := HashContent(data1)

	expectedHashRaw := sha256.Sum256(data1)
	expectedHashHex := hex.EncodeToString(expectedHashRaw[:])
	assert.Equal(t, expectedHashHex, hash1, "Hash of 'hello world' should match expected")

	data2 := []byte("another test string")
	hash2 := HashContent(data2)
	assert.NotEqual(t, hash1, hash2, "Hashes of different content should be different")
	assert.NotEmpty(t, hash2, "Hash should not be empty")

	emptyData := []byte{}
	emptyHash := HashContent(emptyData)
	expectedEmptyHashRaw := sha256.Sum256(emptyData)
	expectedEmptyHashHex := hex.EncodeToString(expectedEmptyHashRaw[:])
	assert.Equal(t, expectedEmptyHashHex, emptyHash, "Hash of empty data should match expected")
}

func TestGetDeterministicInput(t *testing.T) {
	tx1 := Transaction{
		ID:              "id1", // This will be ignored by GetDeterministicInput logic as ID is set from its output
		Timestamp:       1234567890,
		SenderPublicKey: "senderA",
		Type:            TxTypePostCreated,
		Payload:         []byte("payload1"),
		Signature:       []byte("sig1"), // Signature should be excluded
	}
	tx1Output := GetDeterministicInput(tx1)

	// Verify that signature is not in the output
	var tempTx Transaction
	err := json.Unmarshal(tx1Output, &tempTx)
	require.NoError(t, err, "Should be able to unmarshal deterministic input")
	assert.Nil(t, tempTx.Signature, "Signature should be nil in the marshaled output for hashing")
	assert.Equal(t, tx1.Timestamp, tempTx.Timestamp) // Other fields should match

	// Test determinism: same input struct should produce same output []byte
	tx1Again := Transaction{
		Timestamp:       1234567890,
		SenderPublicKey: "senderA",
		Type:            TxTypePostCreated,
		Payload:         []byte("payload1"),
		Signature:       []byte("another_sig_should_not_matter"),
	}
	tx1AgainOutput := GetDeterministicInput(tx1Again)
	assert.Equal(t, tx1Output, tx1AgainOutput, "Deterministic output should be same for same content (excluding sig)")

	// Test different content produces different output
	tx2 := Transaction{
		Timestamp:       9876543210, // Different timestamp
		SenderPublicKey: "senderB",
		Type:            TxTypeCommentAdded,
		Payload:         []byte("payload2"),
		Signature:       []byte("sig2"),
	}
	tx2Output := GetDeterministicInput(tx2)
	assert.NotEqual(t, tx1Output, tx2Output, "Different transaction content should produce different deterministic output")

	// Test that field order in Go struct definition results in consistent JSON for hashing
	// (json.Marshal for structs is deterministic based on field order)
	// This is implicitly tested by comparing tx1Output and tx1AgainOutput.
}

func TestCalculateMerkleRoot(t *testing.T) {
	// Test case 1: Empty transactions list
	merkleRootEmpty := CalculateMerkleRoot([]Transaction{})
	expectedEmptyMerkleRoot := HashContent([]byte{})
	assert.Equal(t, expectedEmptyMerkleRoot, merkleRootEmpty, "Merkle root of empty list should be hash of empty data")

	// Test case 2: Single transaction
	tx1, _ := NewTransaction("sender1", TxTypePostCreated, []byte("tx1 payload"))
	merkleRootSingle := CalculateMerkleRoot([]Transaction{tx1})
	// For a single transaction, Merkle root is its ID (hash), after sorting (which doesn't change a single item)
	assert.Equal(t, tx1.ID, merkleRootSingle, "Merkle root of single transaction should be its ID")

	// Test case 3: Two transactions
	tx2, _ := NewTransaction("sender2", TxTypeCommentAdded, []byte("tx2 payload"))

	// Ensure IDs are different for the test
	if tx1.ID == tx2.ID {
		// Highly unlikely, but if timestamp resolution isn't enough, add a small delay
		time.Sleep(1 * time.Nanosecond)
		tx2, _ = NewTransaction("sender2", TxTypeCommentAdded, []byte("tx2 payload slightly different"))
	}
	require.NotEqual(t, tx1.ID, tx2.ID, "Transaction IDs must be different for this test")

	hashesTwo := []string{tx1.ID, tx2.ID}
	sort.Strings(hashesTwo) // CalculateMerkleRoot sorts internally, so we sort here for expected value
	expectedMerkleRootTwo := HashContent([]byte(hashesTwo[0] + hashesTwo[1]))
	merkleRootTwo := CalculateMerkleRoot([]Transaction{tx1, tx2})
	assert.Equal(t, expectedMerkleRootTwo, merkleRootTwo, "Merkle root of two transactions")

	// Test case 4: Three transactions (odd number, last one duplicated)
	tx3, _ := NewTransaction("sender3", TxTypeLike, []byte("tx3 payload"))
	require.NotEqual(t, tx1.ID, tx3.ID)
	require.NotEqual(t, tx2.ID, tx3.ID)

	hashesThree := []string{tx1.ID, tx2.ID, tx3.ID}
	sort.Strings(hashesThree) // Initial sorted list of tx IDs

	// Level 1 (after potential duplication for odd number)
	// hashesThree will be [h1, h2, h3, h3] after duplication inside CalculateMerkleRoot logic
	// (assuming h1, h2, h3 is the sorted order of tx1.ID, tx2.ID, tx3.ID)
	level1_hash1 := HashContent([]byte(hashesThree[0] + hashesThree[1]))
	level1_hash2 := HashContent([]byte(hashesThree[2] + hashesThree[2])) // tx3.ID duplicated

	// Level 2 (final root)
	// Need to sort level1_hash1 and level1_hash2 if their order isn't guaranteed by input.
	// However, CalculateMerkleRoot processes pairs as they are (after initial overall sort of IDs).
	// The internal loop processes pairs: (h1,h2), (h3,h3_dup).
	// Let's trace CalculateMerkleRoot's logic:
	// Initial: [tx1.ID, tx2.ID, tx3.ID] -> sorted: [ID_s1, ID_s2, ID_s3]
	// Odd, so duplicated: [ID_s1, ID_s2, ID_s3, ID_s3]
	// newHashes level 1: [Hash(ID_s1+ID_s2), Hash(ID_s3+ID_s3)]
	// newHashes level 2: [Hash( Hash(ID_s1+ID_s2) + Hash(ID_s3+ID_s3) )] -> assuming Hash(ID_s1+ID_s2) comes before Hash(ID_s3+ID_s3) if they were sorted
	// The current CalculateMerkleRoot does *not* sort intermediate levels, it processes them in order.

	// Recalculate based on the implementation:
	sortedIDs := []string{tx1.ID, tx2.ID, tx3.ID}
	sort.Strings(sortedIDs)
	h_s1_s2 := HashContent([]byte(sortedIDs[0] + sortedIDs[1]))
	h_s3_s3 := HashContent([]byte(sortedIDs[2] + sortedIDs[2])) // tx3 duplicated with itself

	// The order of h_s1_s2 and h_s3_s3 for the final combination depends on their string values
	// if they were to be sorted again. But CalculateMerkleRoot doesn't re-sort levels.
	// It just takes the newHashes list as is for the next iteration.
	// So, the combined string for the root will be h_s1_s2 + h_s3_s3 (in that order of generation)
	expectedMerkleRootThree := HashContent([]byte(h_s1_s2 + h_s3_s3))

	merkleRootThree := CalculateMerkleRoot([]Transaction{tx1, tx2, tx3})
	assert.Equal(t, expectedMerkleRootThree, merkleRootThree, "Merkle root of three transactions")

	// Test case 5: Four transactions (even number)
	tx4, _ := NewTransaction("sender4", TxTypeProfileUpdated, []byte("tx4 payload"))
	require.NotEmpty(t, tx4.ID)

	merkleRootFour := CalculateMerkleRoot([]Transaction{tx1, tx2, tx3, tx4})
	assert.NotEmpty(t, merkleRootFour, "Merkle root of four transactions should not be empty")
	// More detailed check for 4 tx:
	sortedIDsFour := []string{tx1.ID, tx2.ID, tx3.ID, tx4.ID}
	sort.Strings(sortedIDsFour)
	l1_h1 := HashContent([]byte(sortedIDsFour[0] + sortedIDsFour[1]))
	l1_h2 := HashContent([]byte(sortedIDsFour[2] + sortedIDsFour[3]))
	expectedMerkleRootFour := HashContent([]byte(l1_h1+l1_h2)) // Assuming l1_h1 and l1_h2 are combined in this order
	assert.Equal(t, expectedMerkleRootFour, merkleRootFour, "Merkle root of four transactions")


	// Test that order of transactions in input slice doesn't change Merkle root (due to initial sort of IDs)
	merkleRootDisordered := CalculateMerkleRoot([]Transaction{tx3, tx1, tx4, tx2})
	assert.Equal(t, expectedMerkleRootFour, merkleRootDisordered, "Merkle root should be consistent regardless of initial tx order")

	// Test with a transaction that has an empty ID (should panic, as per added check)
	txWithEmptyID := Transaction{ID: "", Timestamp: 1, SenderPublicKey: "s", Type: TxTypePostCreated, Payload: []byte("p")}
	assert.PanicsWithValue(t, "Transaction in CalculateMerkleRoot has empty ID", func() {
		CalculateMerkleRoot([]Transaction{txWithEmptyID})
	}, "Should panic if a transaction has an empty ID")
}

func TestGetDeterministicInput_Panic(t *testing.T) {
    // Test case where json.Marshal would fail
    // A common way to make json.Marshal fail is with a channel or function type field,
    // but Transaction struct doesn't have those.
    // Another way is if Payload contains something unmarshalable and we try to unmarshal it
    // within GetDeterministicInput (which we don't).
    // The current Transaction struct with basic types and []byte should always marshal.
    // To simulate a panic, we'd have to modify Transaction or introduce a type that causes issues.

    // For now, assume valid Transaction structure that always marshals.
    // If a specific scenario for panic is identified, it can be added.
    // The panic in GetDeterministicInput is more of a safeguard for unexpected errors.
    t.Skip("Skipping panic test for GetDeterministicInput as current Transaction struct is always marshalable.")
}
