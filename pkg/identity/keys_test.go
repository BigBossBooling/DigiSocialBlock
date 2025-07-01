// pkg/identity/keys_test.go
package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateECDSAKeyPair(t *testing.T) {
	privKey, pubKey, err := GenerateECDSAKeyPair()
	require.NoError(t, err, "GenerateECDSAKeyPair should not return an error")
	require.NotNil(t, privKey, "Generated private key should not be nil")
	require.NotNil(t, pubKey, "Generated public key should not be nil")

	// Check if the public key matches the one derived from the private key
	assert.Equal(t, &privKey.PublicKey, pubKey, "Returned public key should match private key's public key")

	// Basic validation of the key properties (e.g., curve)
	assert.Equal(t, elliptic.P256(), privKey.Curve, "Private key should use P256 curve")
	assert.Equal(t, elliptic.P256(), pubKey.Curve, "Public key should use P256 curve")
	assert.True(t, privKey.IsOnCurve(privKey.X, privKey.Y), "Private key's public components should be on the curve")
    assert.True(t, pubKey.IsOnCurve(pubKey.X, pubKey.Y), "Public key should be on the curve")

}

func TestKeySerializationDeserialization(t *testing.T) {
	// 1. Test Private Key Serialization/Deserialization
	privKey1, _, err := GenerateECDSAKeyPair()
	require.NoError(t, err)

	privKeyBytes, err := PrivateKeyToBytes(privKey1)
	require.NoError(t, err, "PrivateKeyToBytes should not error")
	require.NotEmpty(t, privKeyBytes, "Serialized private key bytes should not be empty")

	privKey2, err := BytesToPrivateKey(privKeyBytes)
	require.NoError(t, err, "BytesToPrivateKey should not error")
	require.NotNil(t, privKey2, "Deserialized private key should not be nil")
	assert.True(t, privKey1.Equal(privKey2), "Original and deserialized private keys should be equal")

	// Test hex string conversion for private key
	privKeyHexString, err := PrivateKeyToHexString(privKey1)
	require.NoError(t, err)
	require.NotEmpty(t, privKeyHexString)

	privKey3, err := HexStringToPrivateKey(privKeyHexString)
	require.NoError(t, err)
	require.NotNil(t, privKey3)
	assert.True(t, privKey1.Equal(privKey3), "Original and hex-deserialized private keys should be equal")


	// 2. Test Public Key Serialization/Deserialization
	_, pubKey1, err := GenerateECDSAKeyPair() // Generate a new pair for public key part
	require.NoError(t, err)

	pubKeyBytes, err := PublicKeyToBytes(pubKey1)
	require.NoError(t, err, "PublicKeyToBytes should not error")
	require.NotEmpty(t, pubKeyBytes, "Serialized public key bytes should not be empty")

	pubKey2, err := BytesToPublicKey(pubKeyBytes)
	require.NoError(t, err, "BytesToPublicKey should not error")
	require.NotNil(t, pubKey2, "Deserialized public key should not be nil")
	assert.True(t, pubKey1.Equal(pubKey2), "Original and deserialized public keys should be equal")

	// Test hex string conversion for public key
	pubKeyHexString, err := PublicKeyToHexString(pubKey1)
	require.NoError(t, err)
	require.NotEmpty(t, pubKeyHexString)

	pubKey3, err := HexStringToPublicKey(pubKeyHexString)
	require.NoError(t, err)
	require.NotNil(t, pubKey3)
	assert.True(t, pubKey1.Equal(pubKey3), "Original and hex-deserialized public keys should be equal")


	// Error conditions
	_, err = PrivateKeyToBytes(nil)
	assert.Error(t, err, "PrivateKeyToBytes with nil key should error")
	_, err = BytesToPrivateKey(nil)
	assert.Error(t, err, "BytesToPrivateKey with nil bytes should error")
	_, err = BytesToPrivateKey([]byte("invalid der"))
	assert.Error(t, err, "BytesToPrivateKey with invalid DER should error")

	_, err = PublicKeyToBytes(nil)
	assert.Error(t, err, "PublicKeyToBytes with nil key should error")
	_, err = BytesToPublicKey(nil)
	assert.Error(t, err, "BytesToPublicKey with nil bytes should error")
	_, err = BytesToPublicKey([]byte("invalid der public"))
	assert.Error(t, err, "BytesToPublicKey with invalid DER should error")

	// Test BytesToPublicKey with a non-ECDSA public key type (if possible to create such DER bytes easily)
    // For now, this is covered by the "parsed key is not an ECDSA public key" error check.
}

func TestPublicKeyToAddress(t *testing.T) {
	_, pubKey, err := GenerateECDSAKeyPair()
	require.NoError(t, err)

	address, err := PublicKeyToAddress(pubKey)
	require.NoError(t, err, "PublicKeyToAddress should not error for valid key")
	require.NotEmpty(t, address, "Address string should not be empty")

	// Verify it's a hex string
	assert.Regexp(t, `^[0-9a-fA-F]+$`, address, "Address should be a hex string")

	// Test consistency: same public key should always produce same address
	addressAgain, err := PublicKeyToAddress(pubKey)
	require.NoError(t, err)
	assert.Equal(t, address, addressAgain, "Address generation should be deterministic for the same key")

	// Test with different public key
	_, pubKey2, err := GenerateECDSAKeyPair()
	require.NoError(t, err)
	address2, err := PublicKeyToAddress(pubKey2)
	require.NoError(t, err)
	assert.NotEqual(t, address, address2, "Addresses for different public keys should be different")

	// Error condition
	_, err = PublicKeyToAddress(nil)
	assert.Error(t, err, "PublicKeyToAddress with nil key should error")

	// Check if the address matches the hex of PublicKeyToBytes output
	pubKeyBytes, _ := PublicKeyToBytes(pubKey)
	assert.Equal(t, hex.EncodeToString(pubKeyBytes), address, "Address should be hex of PublicKeyToBytes")
}
