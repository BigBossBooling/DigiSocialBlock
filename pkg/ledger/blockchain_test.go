// pkg/ledger/blockchain_test.go
package ledger

import (
	"encoding/json" // Added for json.Marshal
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create some valid placeholder transactions for blockchain tests
func createBlockchainTestTransactions(t *testing.T, count int, uniquePayloadPrefix string) []Transaction {
	transactions := make([]Transaction, count)
	for i := 0; i < count; i++ {
		payload := fmt.Sprintf("%s_payload_%d", uniquePayloadPrefix, i)
		tx, err := NewTransaction("senderForBlockchainTest", TxTypePostCreated, []byte(payload))
		require.NoError(t, err)
		err = tx.Sign([]byte("dummyKey")) // Sign with placeholder
		require.NoError(t, err)
		transactions[i] = tx
		// Ensure unique timestamps if NewTransaction relies on it for ID uniqueness,
		// especially if tests run fast or `count` is high in a loop.
		if count > 1 && i < count-1 {
			time.Sleep(1 * time.Nanosecond)
		}
	}
	return transactions
}

func TestNewBlockchain(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err, "NewBlockchain should not return an error")
	require.NotNil(t, bc, "Blockchain object should not be nil")
	require.Len(t, bc.Blocks, 1, "Blockchain should have 1 block (genesis block) upon creation")

	genesisBlock := bc.Blocks[0]
	assert.Equal(t, int64(0), genesisBlock.Index, "Genesis block index should be 0")
	assert.Empty(t, genesisBlock.PrevBlockHash, "Genesis block PrevBlockHash should be empty")
	assert.Empty(t, genesisBlock.Transactions, "Genesis block should have no transactions")
	assert.NotEmpty(t, genesisBlock.Hash, "Genesis block hash should not be empty")
	assert.True(t, genesisBlock.ValidateSelf(), "Genesis block should be self-valid")
}

func TestBlockchain_AddBlock(t *testing.T) {
	bc, _ := NewBlockchain()
	initialBlockCount := len(bc.Blocks)

	transactions1 := createBlockchainTestTransactions(t, 2, "block1")
	newBlock1, err := bc.AddBlock(transactions1)
	require.NoError(t, err, "AddBlock should not error for valid transactions")
	assert.Len(t, bc.Blocks, initialBlockCount+1, "Blockchain should have one more block after AddBlock")

	latestBlock, _ := bc.GetLatestBlock()
	assert.Equal(t, newBlock1.Hash, latestBlock.Hash, "Latest block should be the newly added block")
	assert.Equal(t, int64(initialBlockCount), newBlock1.Index, "New block index should be correct")
	assert.Equal(t, bc.Blocks[initialBlockCount-1].Hash, newBlock1.PrevBlockHash, "New block's PrevBlockHash should match previous block's hash")
	assert.Equal(t, transactions1, newBlock1.Transactions, "New block's transactions should match input")
	assert.True(t, newBlock1.ValidateSelf(), "Newly added block should be self-valid")

	// Add another block
	transactions2 := createBlockchainTestTransactions(t, 1, "block2")
	newBlock2, err := bc.AddBlock(transactions2)
	require.NoError(t, err)
	assert.Len(t, bc.Blocks, initialBlockCount+2)
	assert.Equal(t, newBlock1.Hash, newBlock2.PrevBlockHash)
	assert.True(t, newBlock2.ValidateSelf())

	// Test AddBlock with invalid transaction (placeholder VerifySignature fails)
	invalidTx, _ := NewTransaction("senderInvalid", TxTypeCommentAdded, []byte("invalid tx payload"))
	// Do not sign, or tamper signature
	invalidTx.Signature = []byte("bad_sig")
	assert.False(t, invalidTx.VerifySignature())

	_, err = bc.AddBlock([]Transaction{invalidTx})
	assert.Error(t, err, "AddBlock should error if a transaction is invalid")
	assert.Len(t, bc.Blocks, initialBlockCount+2, "Blockchain length should not change after failed AddBlock")
}

func TestBlockchain_GetLatestBlock(t *testing.T) {
	bc, _ := NewBlockchain()

	// Test with genesis block
	latest, err := bc.GetLatestBlock()
	require.NoError(t, err)
	assert.Equal(t, bc.Blocks[0].Hash, latest.Hash)

	// Add a block and test again
	bc.AddBlock(createBlockchainTestTransactions(t, 1, "latestTest"))
	latestAfterAdd, err := bc.GetLatestBlock()
	require.NoError(t, err)
	assert.Equal(t, bc.Blocks[1].Hash, latestAfterAdd.Hash)
	assert.Equal(t, int64(1), latestAfterAdd.Index)
}

func TestBlockchain_GetBlockByIndex(t *testing.T) {
    bc, _ := NewBlockchain()
    bc.AddBlock(createBlockchainTestTransactions(t, 1, "block1")) // Block 1
    bc.AddBlock(createBlockchainTestTransactions(t, 1, "block2")) // Block 2

    // Test valid indices
    block0, err := bc.GetBlockByIndex(0)
    assert.NoError(t, err)
    assert.Equal(t, bc.Blocks[0].Hash, block0.Hash)

    block1, err := bc.GetBlockByIndex(1)
    assert.NoError(t, err)
    assert.Equal(t, bc.Blocks[1].Hash, block1.Hash)

    block2, err := bc.GetBlockByIndex(2)
    assert.NoError(t, err)
    assert.Equal(t, bc.Blocks[2].Hash, block2.Hash)

    // Test out of range
    _, err = bc.GetBlockByIndex(-1)
    assert.Error(t, err, "Should error for negative index")

    _, err = bc.GetBlockByIndex(3) // Current length is 3 (0, 1, 2)
    assert.Error(t, err, "Should error for index out of current max range")

    _, err = bc.GetBlockByIndex(int64(len(bc.Blocks)))
    assert.Error(t, err, "Should error for index equal to length")
}


func TestIsChainValid(t *testing.T) {
	bc, _ := NewBlockchain()

	// Case 1: Valid chain with genesis block only
	isValid, err := bc.IsChainValid()
	assert.NoError(t, err)
	assert.True(t, isValid, "Chain with only genesis block should be valid")

	// Case 2: Valid chain with multiple blocks
	bc.AddBlock(createBlockchainTestTransactions(t, 2, "chainvalid_b1"))
	bc.AddBlock(createBlockchainTestTransactions(t, 1, "chainvalid_b2"))
	isValid, err = bc.IsChainValid()
	assert.NoError(t, err)
	assert.True(t, isValid, "Chain with multiple valid blocks should be valid")

	// Case 3: Chain with a block that fails self-validation (tampered hash)
	bc.Blocks[1].Hash = "tamperedHashInChain"
	isValid, err = bc.IsChainValid()
	assert.Error(t, err, "IsChainValid should return an error for tampered hash")
	assert.False(t, isValid, "Chain with tampered block hash should be invalid")
	assert.Contains(t, err.Error(), "failed self-validation", "Error message should indicate self-validation failure")
	bc, _ = NewBlockchain() // Reset for next test
	bc.AddBlock(createBlockchainTestTransactions(t, 1, "reset_b1"))


	// Case 4: Chain with PrevBlockHash mismatch
	bc.AddBlock(createBlockchainTestTransactions(t, 1, "prevhash_b2"))
	originalPrevHash := bc.Blocks[2].PrevBlockHash
	bc.Blocks[2].PrevBlockHash = "incorrectPrevHashLink"
	isValid, err = bc.IsChainValid()
	assert.Error(t, err)
	assert.False(t, isValid, "Chain with PrevBlockHash mismatch should be invalid")
	assert.Contains(t, err.Error(), "PrevBlockHash", "Error message should indicate prev hash mismatch")
	// Restore for next check, or reset bc
	bc.Blocks[2].PrevBlockHash = originalPrevHash
	// Re-validate self as PrevBlockHash change affects block's content for hashing if not careful,
	// but ValidateSelf only checks its own hash based on its content (excluding its own hash field).
	// The IsChainValid loop checks ValidateSelf then PrevBlockHash link.
	// For this test, ensure the block itself is valid before testing the link.
	// The current ValidateSelf will pass as long as its Hash field matches its content.
	// The PrevBlockHash is part of that content. So if we only change PrevBlockHash field
	// but not its Hash field, ValidateSelf will fail.
	// Let's reset and construct a more precise scenario for PrevBlockHash mismatch.

	bc, _ = NewBlockchain()
	b1tx := createBlockchainTestTransactions(t,1,"b1pv")
	block1, _ := bc.AddBlock(b1tx)
	b2tx := createBlockchainTestTransactions(t,1,"b2pv")
	block2, _ := bc.AddBlock(b2tx) // block2.PrevBlockHash is block1.Hash

	block2TamperedLink := block2
	block2TamperedLink.PrevBlockHash = "totallyWrongPreviousHash"
	// To make this block *itself* valid with this tampered link, its own hash would need recalculation.
	// But we are testing IsChainValid's check of the *link*, not just self-validation.
	// If block2TamperedLink.ValidateSelf() is called, it would fail if its .Hash field
	// doesn't match its content (which now includes the wrong PrevBlockHash).
	// Let's assume block2TamperedLink is put into the chain as is.
	bc.Blocks[2] = block2TamperedLink // block2 is at index 2

	// If block2TamperedLink.Hash was NOT updated, ValidateSelf for Blocks[2] will fail first.
	// This is because HashContent(block2TamperedLink without .Hash) will be different.
	// To specifically test the PrevBlockHash link check in IsChainValid, Blocks[2] must pass ValidateSelf.
	// So, we must ensure block2TamperedLink.Hash is "correct" for its content including the wrong PrevBlockHash.

	blockForRecalc := block2TamperedLink
	blockForRecalc.Hash = ""
	jsonData, _ := json.Marshal(blockForRecalc)
	block2TamperedLink.Hash = HashContent(jsonData)
	bc.Blocks[2] = block2TamperedLink
	require.True(t, bc.Blocks[2].ValidateSelf(), "Tampered link block must be self-valid for this specific test part")

	isValid, err = bc.IsChainValid()
	assert.Error(t, err, "IsChainValid should error for prev hash mismatch")
	assert.False(t, isValid, "Chain with explicit PrevBlockHash mismatch should be invalid")
	assert.Contains(t, err.Error(), "does not match previous block's hash", "Error message should specify prev hash mismatch")


	// Case 5: Chain with a block containing an invalid transaction (placeholder signature)
	bc, _ = NewBlockchain()
	bc.AddBlock(createBlockchainTestTransactions(t, 1, "validtx_b1"))
	validTx := createBlockchainTestTransactions(t, 1, "validtx_b2_tx1")[0]
	invalidTxInBlock, _ := NewTransaction("senderInvalidSig", TxTypePostCreated, []byte("payloadInvalidSig"))
	invalidTxInBlock.Signature = []byte("thisisnottheplaceholdersigfor" + invalidTxInBlock.ID) // Ensure VerifySignature fails

	latestBlockForSigTest, errGetLatest := bc.GetLatestBlock()
	require.NoError(t, errGetLatest)

	// Attempt to create a block with an invalid transaction. NewBlock should prevent this.
	_, errCreatingBlockWithInvalidTx := NewBlock(latestBlockForSigTest.Index+1, latestBlockForSigTest.Hash, []Transaction{validTx, invalidTxInBlock})
	assert.Error(t, errCreatingBlockWithInvalidTx, "NewBlock should fail if a transaction's signature is invalid")

	// To specifically test IsChainValid's check for invalid transactions (if one somehow got in),
	// we manually craft and insert such a block.
	craftedBlock := Block{
		Index: latestBlockForSigTest.Index + 1,
		Timestamp: time.Now().UnixNano(),
		Transactions: []Transaction{validTx, invalidTxInBlock}, // invalidTxInBlock.VerifySignature() is false
		PrevBlockHash: latestBlockForSigTest.Hash,
	}
	craftedBlock.MerkleRoot = CalculateMerkleRoot(craftedBlock.Transactions)
	blockForHashCrafted := craftedBlock
	blockForHashCrafted.Hash = ""
	jsonCrafted, errJsonMarshalCrafted := json.Marshal(blockForHashCrafted)
	require.NoError(t, errJsonMarshalCrafted)
	craftedBlock.Hash = HashContent(jsonCrafted)
	require.True(t, craftedBlock.ValidateSelf(), "Crafted block must be self-valid for this test part") // It is self-valid in its structure

	bc.Blocks = append(bc.Blocks, craftedBlock) // Manually append the block with the bad transaction
	isValid, err = bc.IsChainValid()
	assert.Error(t, err, "IsChainValid should error for invalid tx sig in manually appended block")
	assert.False(t, isValid, "Chain with invalid transaction signature should be invalid")
	assert.Contains(t, err.Error(), "invalid signature", "Error message should specify invalid signature")
}

func TestBlockchain_GetTransactionsInBlock(t *testing.T) {
    bc, _ := NewBlockchain()
    txsB1 := createBlockchainTestTransactions(t, 2, "tx_b1")
    bc.AddBlock(txsB1)
    txsB2 := createBlockchainTestTransactions(t, 1, "tx_b2")
    bc.AddBlock(txsB2)

    // Get from genesis (no transactions)
    retrievedTxsGenesis, err := bc.GetTransactionsInBlock(0)
    assert.NoError(t, err)
    assert.Empty(t, retrievedTxsGenesis)

    // Get from block 1
    retrievedTxsB1, err := bc.GetTransactionsInBlock(1)
    assert.NoError(t, err)
    assert.Equal(t, txsB1, retrievedTxsB1)

    // Get from block 2
    retrievedTxsB2, err := bc.GetTransactionsInBlock(2)
    assert.NoError(t, err)
    assert.Equal(t, txsB2, retrievedTxsB2)

    // Error cases
    _, err = bc.GetTransactionsInBlock(-1)
    assert.Error(t, err)
    _, err = bc.GetTransactionsInBlock(int64(len(bc.Blocks))) // Out of bounds
    assert.Error(t, err)
}

// Test concurrent AddBlock calls (basic safety check for mutex)
func TestBlockchain_AddBlock_Concurrent(t *testing.T) {
    bc, _ := NewBlockchain()
    numGoroutines := 5
    blocksToAddPerGoroutine := 2
    var wg sync.WaitGroup

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(goroutineID int) {
            defer wg.Done()
            for j := 0; j < blocksToAddPerGoroutine; j++ {
                payloadPrefix := fmt.Sprintf("g%d_b%d", goroutineID, j)
                txs := createBlockchainTestTransactions(t, 1, payloadPrefix)
                // Add a small delay to increase chance of interleaving, though global lock makes it less critical
                // time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
                _, err := bc.AddBlock(txs)
                // The global lock in AddBlock should prevent most race conditions leading to errors here.
                // If AddBlock were more complex (e.g. releasing lock mid-operation), this could fail.
                assert.NoError(t, err, fmt.Sprintf("goroutine %d, block %d add failed", goroutineID, j))
            }
        }(i)
    }
    wg.Wait()

    expectedBlockCount := 1 /* genesis */ + (numGoroutines * blocksToAddPerGoroutine)
    assert.Len(t, bc.Blocks, expectedBlockCount, "Blockchain should have correct number of blocks after concurrent adds")

    isValid, err := bc.IsChainValid()
    assert.NoError(t, err)
    assert.True(t, isValid, "Blockchain should remain valid after concurrent adds")
}
