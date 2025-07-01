// pkg/identity/wallet_test.go
package identity

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWallet(t *testing.T) {
	wallet, err := NewWallet()
	require.NoError(t, err, "NewWallet should not return an error")
	require.NotNil(t, wallet, "Wallet should not be nil")
	require.NotNil(t, wallet.PrivateKey, "Wallet's private key should not be nil")

	// Check if public key can be derived
	pubKey := wallet.GetPublicKey()
	require.NotNil(t, pubKey, "Derived public key should not be nil")
	assert.True(t, pubKey.IsOnCurve(pubKey.X, pubKey.Y), "Public key should be valid")
}

func TestWallet_GetAddress(t *testing.T) {
	wallet, _ := NewWallet()
	address, err := wallet.GetAddress()
	require.NoError(t, err, "GetAddress should not return an error")
	require.NotEmpty(t, address, "Address should not be empty")

	// Compare with directly generating address from public key
	expectedAddr, _ := PublicKeyToAddress(wallet.GetPublicKey())
	assert.Equal(t, expectedAddr, address, "Wallet address should match address from its public key")

	// Wallet with nil private key
	nilKeyWallet := &Wallet{PrivateKey: nil}
	_, err = nilKeyWallet.GetAddress()
	assert.Error(t, err, "GetAddress should error if private key is nil")
}

func TestWallet_SignAndVerify(t *testing.T) {
	wallet, _ := NewWallet()
	dataToSign := []byte("This is some data to be signed by the wallet.")

	// Hash the data first, as Sign expects a hash
	dataHash := sha256.Sum256(dataToSign)

	signature, err := wallet.Sign(dataHash[:])
	require.NoError(t, err, "Sign method should not return an error")
	require.NotEmpty(t, signature, "Signature should not be empty")

	// Verify the signature using the wallet's public key
	pubKey := wallet.GetPublicKey()
	require.NotNil(t, pubKey)
	isValid := ecdsa.VerifyASN1(pubKey, dataHash[:], signature)
	assert.True(t, isValid, "Signature should be verifiable with the wallet's public key")

	// Test with different data (should not verify)
	differentDataHash := sha256.Sum256([]byte("Different data"))
	isInvalid := ecdsa.VerifyASN1(pubKey, differentDataHash[:], signature)
	assert.False(t, isInvalid, "Signature should not verify for different data hash")

	// Test with different public key (should not verify)
	wallet2, _ := NewWallet()
	pubKey2 := wallet2.GetPublicKey()
	isInvalid2 := ecdsa.VerifyASN1(pubKey2, dataHash[:], signature)
	assert.False(t, isInvalid2, "Signature should not verify with a different public key")

	// Error case for Sign: nil private key
	nilKeyWallet := &Wallet{PrivateKey: nil}
	_, err = nilKeyWallet.Sign(dataHash[:])
	assert.Error(t, err, "Sign should error if private key is nil")

	// Error case for Sign: empty data hash
	_, err = wallet.Sign([]byte{})
	assert.Error(t, err, "Sign should error for empty data hash")

}

func TestWallet_SignData(t *testing.T) {
	wallet, _ := NewWallet()
	rawData := []byte("Raw data to test SignData")

	signature, err := wallet.SignData(rawData)
	require.NoError(t, err)
	require.NotEmpty(t, signature)

	// Verify
	expectedHash := sha256.Sum256(rawData)
	isValid := ecdsa.VerifyASN1(wallet.GetPublicKey(), expectedHash[:], signature)
	assert.True(t, isValid, "Signature from SignData should be valid")

	// Error case: empty raw data
	_, err = wallet.SignData([]byte{})
	assert.Error(t, err)
}


func TestWallet_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	walletFilepath := filepath.Join(tempDir, "test_wallet.json")

	// 1. Create a new wallet and save it
	originalWallet, err := NewWallet()
	require.NoError(t, err)
	require.NotNil(t, originalWallet.PrivateKey)

	originalAddress, _ := originalWallet.GetAddress()

	err = originalWallet.SaveToFile(walletFilepath)
	require.NoError(t, err, "SaveToFile should not return an error")

	// 2. Load the wallet from the file
	loadedWallet, err := LoadWalletFromFile(walletFilepath)
	require.NoError(t, err, "LoadWalletFromFile should not return an error")
	require.NotNil(t, loadedWallet, "Loaded wallet should not be nil")
	require.NotNil(t, loadedWallet.PrivateKey, "Loaded wallet's private key should not be nil")

	// 3. Verify the loaded wallet is the same as the original
	assert.True(t, originalWallet.PrivateKey.Equal(loadedWallet.PrivateKey), "Original and loaded private keys should be equal")

	loadedAddress, _ := loadedWallet.GetAddress()
	assert.Equal(t, originalAddress, loadedAddress, "Address of original and loaded wallet should be the same")

	// 4. Test signing with the loaded wallet to ensure functionality
	dataToSign := []byte("data for loaded wallet")
	dataHash := sha256.Sum256(dataToSign)
	signature, err := loadedWallet.Sign(dataHash[:])
	require.NoError(t, err)
	isValid := ecdsa.VerifyASN1(loadedWallet.GetPublicKey(), dataHash[:], signature)
	assert.True(t, isValid, "Signature from loaded wallet should be valid")

	// Test error conditions for SaveToFile
	err = (&Wallet{}).SaveToFile(filepath.Join(tempDir, "empty_wallet.json")) // No private key
	assert.Error(t, err, "SaveToFile should error if private key is nil")
	// Test invalid path (e.g. directory) - os.WriteFile will handle this
	err = originalWallet.SaveToFile(tempDir) // Saving to a directory path
    assert.Error(t, err, "SaveToFile should error for invalid filepath (directory)")


	// Test error conditions for LoadWalletFromFile
	_, err = LoadWalletFromFile(filepath.Join(tempDir, "non_existent_wallet.json"))
	assert.Error(t, err, "LoadWalletFromFile should error if file does not exist")

	// Create a malformed JSON file
	malformedJsonFile := filepath.Join(tempDir, "malformed_wallet.json")
	err = os.WriteFile(malformedJsonFile, []byte("{not_a_json"), 0600)
	require.NoError(t, err)
	_, err = LoadWalletFromFile(malformedJsonFile)
	assert.Error(t, err, "LoadWalletFromFile should error for malformed JSON")

	// Create JSON with missing private_key_hex
	missingKeyJsonFile := filepath.Join(tempDir, "missingkey_wallet.json")
	err = os.WriteFile(missingKeyJsonFile, []byte(`{"some_other_field":"value"}`), 0600)
	require.NoError(t, err)
	_, err = LoadWalletFromFile(missingKeyJsonFile)
	assert.Error(t, err, "LoadWalletFromFile should error if private_key_hex is missing")


	// Create JSON with invalid private_key_hex
	invalidHexJsonFile := filepath.Join(tempDir, "invalidhex_wallet.json")
	err = os.WriteFile(invalidHexJsonFile, []byte(`{"private_key_hex":"not-a-hex-string"}`), 0600)
	require.NoError(t, err)
	_, err = LoadWalletFromFile(invalidHexJsonFile)
	assert.Error(t, err, "LoadWalletFromFile should error if private_key_hex is invalid hex")

}

func TestWallet_CustomJSONMarshalUnmarshal(t *testing.T) {
	wallet, err := NewWallet()
	require.NoError(t, err)

	jsonData, err := wallet.MarshalJSON()
	require.NoError(t, err)

	// Check if it contains "private_key_hex"
	jsonString := string(jsonData)
	assert.Contains(t, jsonString, `"private_key_hex":`)

	var newWallet Wallet
	err = newWallet.UnmarshalJSON(jsonData)
	require.NoError(t, err)

	assert.True(t, wallet.PrivateKey.Equal(newWallet.PrivateKey), "Wallet after marshal/unmarshal should be equal")

	// Test unmarshal error with bad data
	badJsonData := []byte(`{"private_key_hex":"invalid-hex-for-sure"}`)
	var errorWallet Wallet
	err = errorWallet.UnmarshalJSON(badJsonData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert hex string to private key")

	// Test unmarshal error with totally invalid json
	totallyBadJson := []byte(`{xxxx`)
	err = errorWallet.UnmarshalJSON(totallyBadJson)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal wallet JSON data")

}
