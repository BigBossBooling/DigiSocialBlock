// pkg/identity/wallet.go
package identity

import (
	"crypto/ecdsa"
	"crypto/rand" // Required for ecdsa.Sign
	"crypto/sha256" // For hashing data before signing, if raw data is passed
	"encoding/json"
	"fmt"
	"os"
)

// Wallet stores an ECDSA private key and provides methods for signing and address generation.
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey `json:"private_key"` // Store the private key directly for JSON marshal/unmarshal
}

// walletJSON is used for custom marshaling/unmarshaling of the private key
// as hex string to make the JSON file human-readable and simpler.
type walletJSON struct {
	PrivateKeyHex string `json:"private_key_hex"`
}

// MarshalJSON custom marshaler for Wallet.
func (w Wallet) MarshalJSON() ([]byte, error) {
	if w.PrivateKey == nil {
		return nil, fmt.Errorf("cannot marshal nil private key")
	}
	privKeyHex, err := PrivateKeyToHexString(w.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key to hex for marshaling: %w", err)
	}
	return json.Marshal(walletJSON{
		PrivateKeyHex: privKeyHex,
	})
}

// UnmarshalJSON custom unmarshaler for Wallet.
func (w *Wallet) UnmarshalJSON(data []byte) error {
	var jsonData walletJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal wallet JSON data: %w", err)
	}
	if jsonData.PrivateKeyHex == "" {
		return fmt.Errorf("private_key_hex field is missing or empty in JSON")
	}
	privateKey, err := HexStringToPrivateKey(jsonData.PrivateKeyHex)
	if err != nil {
		return fmt.Errorf("failed to convert hex string to private key during unmarshaling: %w", err)
	}
	w.PrivateKey = privateKey
	return nil
}


// NewWallet generates a new ECDSA key pair and creates a Wallet instance.
func NewWallet() (*Wallet, error) {
	privateKey, _, err := GenerateECDSAKeyPair() // Public key is derived from private
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair for new wallet: %w", err)
	}
	return &Wallet{PrivateKey: privateKey}, nil
}

// GetPublicKey returns the public key corresponding to the wallet's private key.
func (w *Wallet) GetPublicKey() *ecdsa.PublicKey {
	if w.PrivateKey == nil {
		return nil
	}
	return &w.PrivateKey.PublicKey
}

// GetAddress returns the public address/identifier for the wallet.
// This uses PublicKeyToAddress from keys.go.
func (w *Wallet) GetAddress() (string, error) {
	publicKey := w.GetPublicKey()
	if publicKey == nil {
		return "", fmt.Errorf("wallet does not have a private/public key pair")
	}
	return PublicKeyToAddress(publicKey)
}

// Sign takes a hash of data (e.g., a transaction ID, which is already a hash)
// and signs it using the wallet's private key.
// It returns the ASN.1 encoded ECDSA signature.
func (w *Wallet) Sign(dataHash []byte) ([]byte, error) {
	if w.PrivateKey == nil {
		return nil, fmt.Errorf("wallet does not have a private key to sign with")
	}
	if len(dataHash) == 0 {
		return nil, fmt.Errorf("cannot sign empty data hash")
	}
	// ecdsa.SignASN1 automatically handles hashing if the message isn't a hash,
	// but it's common practice to pass the hash directly.
	// The `dataHash` parameter implies the caller has already hashed the data.
	// `ecdsa.SignASN1` takes `rand.Reader`, the private key, and the hash to be signed.
	signature, err := ecdsa.SignASN1(rand.Reader, w.PrivateKey, dataHash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign data hash: %w", err)
	}
	return signature, nil
}

// SignData first hashes the input data using SHA256 then signs the hash.
// This is a convenience function if the caller has raw data.
// Transaction.ID is already a hash, so for ledger.Transaction.Sign, use Wallet.Sign directly.
func (w *Wallet) SignData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot sign empty data")
	}
	hasher := sha256.New()
	hasher.Write(data)
	dataHash := hasher.Sum(nil)
	return w.Sign(dataHash)
}


// SaveToFile saves the wallet (private key) to a JSON file.
// CRUCIAL SECURITY NOTE: In a real-world application, private keys
// MUST BE SECURELY ENCRYPTED AT REST and should never be stored in plain text.
// TODO: Implement encryption for private key before saving.
func (w *Wallet) SaveToFile(filepath string) error {
	if w.PrivateKey == nil {
		return fmt.Errorf("cannot save wallet with no private key")
	}

	jsonData, err := json.MarshalIndent(w, "", "  ") // Use custom MarshalJSON
	if err != nil {
		return fmt.Errorf("failed to marshal wallet to JSON: %w", err)
	}

	// Check if file exists, prompt for overwrite or use a flag? For now, overwrite.
	err = os.WriteFile(filepath, jsonData, 0600) // 0600: read/write for owner only
	if err != nil {
		return fmt.Errorf("failed to write wallet to file '%s': %w", filepath, err)
	}
	return nil
}

// LoadWalletFromFile loads a wallet (private key) from a JSON file.
// Assumes the private key was stored in hex format within the JSON.
func LoadWalletFromFile(filepath string) (*Wallet, error) {
	fileData, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet file '%s': %w", filepath, err)
	}

	var wallet Wallet
	err = json.Unmarshal(fileData, &wallet) // Use custom UnmarshalJSON
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet from JSON data in file '%s': %w", filepath, err)
	}

	if wallet.PrivateKey == nil {
		// This case should be caught by UnmarshalJSON if PrivateKeyHex is missing/empty
		return nil, fmt.Errorf("loaded wallet has no private key from file '%s'", filepath)
	}
	return &wallet, nil
}
