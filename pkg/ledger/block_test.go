// pkg/ledger/block_test.go
package ledger

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a few valid placeholder transactions for block tests
func createTestTransactions(t *testing.T, count int) []Transaction {
	transactions := make([]Transaction, count)
	for i := 0; i < count; i++ {
		tx, err := NewTransaction("senderForBlockTest", TxTypePostCreated, []byte("payload for block test"))
		require.NoError(t, err)
		err = tx.Sign([]byte("dummyKey")) // Sign with placeholder
		require.NoError(t, err)
		transactions[i] = tx
		time.Sleep(1 * time.Nanosecond) // Ensure unique timestamps if NewTransaction relies on it for ID uniqueness
	}
	return transactions
}

func TestNewBlock(t *testing.T) {
	prevBlockHash := HashContent([]byte("previous block's content"))
	transactions := createTestTransactions(t, 2)

	block, err := NewBlock(1, prevBlockHash, transactions)
	require.NoError(t, err, "NewBlock should not error for valid inputs")

	assert.Equal(t, int64(1), block.Index, "Block index should match")
	assert.NotZero(t, block.Timestamp, "Block timestamp should be set")
	assert.Equal(t, transactions, block.Transactions, "Block transactions should match")
	assert.Equal(t, prevBlockHash, block.PrevBlockHash, "Previous block hash should match")

	// Check MerkleRoot calculation
	expectedMerkleRoot := CalculateMerkleRoot(transactions)
	assert.Equal(t, expectedMerkleRoot, block.MerkleRoot, "Merkle root should be correctly calculated")
	assert.NotEmpty(t, block.MerkleRoot, "Merkle root should not be empty for non-empty transactions")

	// Check Block Hash calculation
	assert.NotEmpty(t, block.Hash, "Block hash should be set and not empty")

	blockForHashing := block
	blockForHashing.Hash = ""
	jsonData, _ := json.Marshal(blockForHashing)
	expectedHash := HashContent(jsonData)
	assert.Equal(t, expectedHash, block.Hash, "Block hash should be correctly calculated")

	// Test NewBlock with empty transactions (e.g., genesis block or empty block)
	emptyTransactions := []Transaction{}
	genesisBlock, err := NewBlock(0, "", emptyTransactions)
	require.NoError(t, err)
	assert.Equal(t, int64(0), genesisBlock.Index)
	assert.Equal(t, "", genesisBlock.PrevBlockHash)
	assert.Equal(t, CalculateMerkleRoot(emptyTransactions), genesisBlock.MerkleRoot, "Merkle root for empty tx list")
	assert.NotEmpty(t, genesisBlock.Hash)

	// Test error cases for NewBlock
	_, err = NewBlock(-1, prevBlockHash, transactions)
	assert.Error(t, err, "NewBlock should error for negative index")

	_, err = NewBlock(1, "", transactions) // Non-genesis block with empty prevBlockHash
	assert.Error(t, err, "NewBlock should error for non-genesis block with empty prevBlockHash")

	// Test with invalid transaction (signature verification fails)
	invalidTx, _ := NewTransaction("senderInvalid", TxTypePostCreated, []byte("invalid payload"))
	// Do not sign, or sign then tamper:
	// invalidTx.Sign([]byte("dummyKey"))
	invalidTx.Signature = []byte("clearly_invalid_signature")
	assert.False(t, invalidTx.VerifySignature(), "Test setup: invalidTx should fail VerifySignature")

	_, err = NewBlock(2, prevBlockHash, []Transaction{transactions[0], invalidTx})
	assert.Error(t, err, "NewBlock should error if any transaction is invalid")
}

func TestBlock_ValidateSelf(t *testing.T) {
	// Case 1: Valid block
	transactions := createTestTransactions(t, 1)
	validBlock, err := NewBlock(1, "prevHashValid", transactions)
	require.NoError(t, err)
	assert.True(t, validBlock.ValidateSelf(), "A newly created valid block should pass self-validation")

	// Case 2: Tampered Hash
	tamperedHashBlock := validBlock
	tamperedHashBlock.Hash = "tamperedBlockhash123"
	assert.False(t, tamperedHashBlock.ValidateSelf(), "Block with tampered hash should fail self-validation")

	// Case 3: Tampered MerkleRoot
	tamperedMerkleBlock := validBlock
	// originalMerkleRoot := validBlock.MerkleRoot // This was unused
	tamperedMerkleBlock.MerkleRoot = "tamperedMerkleRootAbc"
	// Recalculate hash because MerkleRoot change affects block's overall hash
	blockForHashing := tamperedMerkleBlock
	blockForHashing.Hash = ""
	jsonData, _ := json.Marshal(blockForHashing)
	tamperedMerkleBlock.Hash = HashContent(jsonData) // Hash is now "correct" for the tampered MerkleRoot
	// ValidateSelf should catch that the MerkleRoot itself is inconsistent with transactions
	assert.False(t, tamperedMerkleBlock.ValidateSelf(), "Block with tampered MerkleRoot (but hash recalculated for it) should fail self-validation because MerkleRoot doesn't match transactions")

    // Case 3.1: MerkleRoot tampered, Hash NOT recalculated (easier to detect)
    tamperedMerkleOnlyBlock := validBlock
    tamperedMerkleOnlyBlock.MerkleRoot = "anotherTamperedMerkleRoot"
    // Hash is still originalBlock.Hash, which was calculated with original MerkleRoot.
    // ValidateSelf first checks MerkleRoot against transactions (will fail here).
    // If that passed, it would check Hash against content (would also fail here).
    assert.False(t, tamperedMerkleOnlyBlock.ValidateSelf(), "Block with only MerkleRoot tampered should fail")


	// Case 4: Tampered Transactions (after block creation, without recalculating MerkleRoot and Hash)
	tamperedTransactionsBlock := validBlock
	if len(tamperedTransactionsBlock.Transactions) > 0 {
		// Modify a transaction slightly. This makes the MerkleRoot and Hash invalid.
		tamperedTransactionsBlock.Transactions[0].Payload = []byte("tampered payload within block")
	}
	assert.False(t, tamperedTransactionsBlock.ValidateSelf(), "Block with tampered transactions (MerkleRoot/Hash not updated) should fail self-validation")

	// Case 5: Empty hash
	emptyHashBlock := validBlock
	emptyHashBlock.Hash = ""
	assert.False(t, emptyHashBlock.ValidateSelf(), "Block with empty hash should fail self-validation")

    // Case 6: Block with no transactions
    noTxBlock, err := NewBlock(2, validBlock.Hash, []Transaction{})
    require.NoError(t, err)
    assert.True(t, noTxBlock.ValidateSelf(), "Block with no transactions should self-validate correctly")
}

func TestBlock_IsValid_Model_vs_ValidateSelf(t *testing.T) {
    // Valid block
	transactions := createTestTransactions(t, 1)
	validBlock, err := NewBlock(1, "prevHashValid", transactions)
	require.NoError(t, err)
    assert.True(t, validBlock.ValidateSelf(), "ValidateSelf should be true for a new valid block")
    assert.True(t, validBlock.IsValid(), "model.IsValid should be true (checks Hash != \"\")")

    // Block with empty hash
    invalidBlock := validBlock
    invalidBlock.Hash = ""
    assert.False(t, invalidBlock.ValidateSelf(), "ValidateSelf should be false if Hash is empty")
    assert.False(t, invalidBlock.IsValid(), "model.IsValid should be false if Hash is empty")
}
