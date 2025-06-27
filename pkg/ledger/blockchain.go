// pkg/ledger/blockchain.go
package ledger

import (
	"fmt"
	"sync"
)

// Blockchain represents the append-only distributed ledger.
type Blockchain struct {
	mu     sync.RWMutex
	Blocks []Block
	// Future: Add fields for difficulty, consensus mechanism, etc.
}

// NewBlockchain creates a new blockchain with a genesis block.
func NewBlockchain() (*Blockchain, error) {
	genesisTransactions := []Transaction{} // Genesis block usually has no operational transactions
	genesisBlock, err := NewBlock(0, "", genesisTransactions)
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis block: %w", err)
	}

	bc := &Blockchain{
		Blocks: []Block{genesisBlock},
	}
	// fmt.Printf("Genesis block created: %s\n", genesisBlock.Hash) // Logging can be handled by the caller
	return bc, nil
}

// AddBlock adds a new block to the blockchain after validation.
func (bc *Blockchain) AddBlock(transactions []Transaction) (Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(bc.Blocks) == 0 {
		return Block{}, fmt.Errorf("blockchain not initialized with a genesis block")
	}
	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	newBlock, err := NewBlock(lastBlock.Index+1, lastBlock.Hash, transactions)
	if err != nil {
		return Block{}, fmt.Errorf("failed to create new block: %w", err)
	}

	// Perform validation before adding
	if !newBlock.ValidateSelf() { // Checks if the new block's hash and Merkle root are correct
		return Block{}, fmt.Errorf("new block (Index: %d) failed self-validation", newBlock.Index)
	}
	if newBlock.PrevBlockHash != lastBlock.Hash {
		return Block{}, fmt.Errorf("new block's (Index: %d) PrevBlockHash %s does not match last block's hash %s",
			newBlock.Index, newBlock.PrevBlockHash, lastBlock.Hash)
	}

	// (Optional but recommended) Further validation: ensure transactions within the block are valid
	// This is already partially done in NewBlock, but could be expanded here for context-specific rules.
	for _, tx := range newBlock.Transactions {
		if !tx.VerifySignature() { // Placeholder check
			return Block{}, fmt.Errorf("transaction %s in new block (Index: %d) has invalid signature", tx.ID, newBlock.Index)
		}
		// Additional transaction validation logic could go here (e.g., double spending checks if applicable)
	}

	bc.Blocks = append(bc.Blocks, newBlock)
	// fmt.Printf("New block added (Index: %d, Hash: %s)\n", newBlock.Index, newBlock.Hash)
	return newBlock, nil
}

// GetLatestBlock returns the latest block in the chain.
func (bc *Blockchain) GetLatestBlock() (Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if len(bc.Blocks) == 0 {
		return Block{}, fmt.Errorf("blockchain is empty")
	}
	return bc.Blocks[len(bc.Blocks)-1], nil
}

// GetBlockByIndex returns a block by its index.
func (bc *Blockchain) GetBlockByIndex(index int64) (Block, error) {
    bc.mu.RLock()
    defer bc.mu.RUnlock()
    if index < 0 || index >= int64(len(bc.Blocks)) {
        return Block{}, fmt.Errorf("block index %d out of range (chain length %d)", index, len(bc.Blocks))
    }
    return bc.Blocks[index], nil
}


// IsChainValid checks the integrity of the entire blockchain.
// Validates block hashes, Merkle roots, and sequential linkage.
func (bc *Blockchain) IsChainValid() (bool, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.Blocks) == 0 {
		return false, fmt.Errorf("cannot validate an empty blockchain")
	}
	if len(bc.Blocks) == 1 { // Only genesis block
		if !bc.Blocks[0].ValidateSelf() {
			return false, fmt.Errorf("genesis block (Index: %d) failed self-validation", bc.Blocks[0].Index)
		}
		return true, nil
	}

	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		// Check self-validation (hash and Merkle root)
		if !currentBlock.ValidateSelf() {
			return false, fmt.Errorf("chain invalid: Block %d failed self-validation", currentBlock.Index)
		}

		// Check previous hash link
		if currentBlock.PrevBlockHash != prevBlock.Hash {
			return false, fmt.Errorf("chain invalid: Block %d PrevBlockHash (%s) does not match previous block's hash (%s)",
				currentBlock.Index, currentBlock.PrevBlockHash, prevBlock.Hash)
		}

		// Check transaction validity within the block (signatures etc.)
		for _, tx := range currentBlock.Transactions {
			if !tx.VerifySignature() { // Placeholder check
				return false, fmt.Errorf("chain invalid: Transaction %s in Block %d has invalid signature", tx.ID, currentBlock.Index)
			}
		}
	}
	return true, nil
}

// GetTransactionsInBlock retrieves all transactions from a given block index.
// Note: This was already defined by the user, but I'm including it for completeness of the file if it wasn't.
// If it was, this would be a duplicate. Assuming it's for completeness here.
func (bc *Blockchain) GetTransactionsInBlock(index int64) ([]Transaction, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if index < 0 || index >= int64(len(bc.Blocks)) {
		return nil, fmt.Errorf("block index %d out of range (chain length %d)", index, len(bc.Blocks))
	}
	return bc.Blocks[index].Transactions, nil
}
