// pkg/identity/keys.go
package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
)

// GenerateECDSAKeyPair generates a new ECDSA private/public key pair using the P-256 curve.
func GenerateECDSAKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// PrivateKeyToBytes serializes an ecdsa.PrivateKey to a byte slice using DER encoding (PKCS#8).
func PrivateKeyToBytes(priv *ecdsa.PrivateKey) ([]byte, error) {
	if priv == nil {
		return nil, fmt.Errorf("private key is nil")
	}
	derBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key to DER: %w", err)
	}
	return derBytes, nil
}

// BytesToPrivateKey deserializes a byte slice (DER encoded PKCS#8) to an ecdsa.PrivateKey.
func BytesToPrivateKey(derBytes []byte) (*ecdsa.PrivateKey, error) {
	if len(derBytes) == 0 {
		return nil, fmt.Errorf("input bytes are empty")
	}
	privateKey, err := x509.ParseECPrivateKey(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DER bytes to private key: %w", err)
	}
	return privateKey, nil
}

// PublicKeyToBytes serializes an ecdsa.PublicKey to a byte slice using DER encoding (PKIX format).
func PublicKeyToBytes(pub *ecdsa.PublicKey) ([]byte, error) {
	if pub == nil {
		return nil, fmt.Errorf("public key is nil")
	}
	derBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key to DER: %w", err)
	}
	return derBytes, nil
}

// BytesToPublicKey deserializes a byte slice (DER encoded PKIX format) to an ecdsa.PublicKey.
func BytesToPublicKey(derBytes []byte) (*ecdsa.PublicKey, error) {
	if len(derBytes) == 0 {
		return nil, fmt.Errorf("input bytes are empty")
	}
	pubAny, err := x509.ParsePKIXPublicKey(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DER bytes to public key: %w", err)
	}
	publicKey, ok := pubAny.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("parsed key is not an ECDSA public key")
	}
	return publicKey, nil
}

// PublicKeyToAddress converts an ecdsa.PublicKey into a unique, concise string identifier.
// For this task, it's a hex-encoded string of the marshaled public key (PKIX, DER format).
func PublicKeyToAddress(pubKey *ecdsa.PublicKey) (string, error) {
	if pubKey == nil {
		return "", fmt.Errorf("public key is nil")
	}
	pubKeyBytes, err := PublicKeyToBytes(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to serialize public key for address generation: %w", err)
	}
	// For an "address", often a hash of the public key is used.
	// However, the task asks for "hex-encoded string of the marshaled public key".
	// This will be longer than a hash but directly represents the key.
	return hex.EncodeToString(pubKeyBytes), nil
}

// Helper function: PrivateKeyToHexString (DER then Hex)
func PrivateKeyToHexString(priv *ecdsa.PrivateKey) (string, error) {
	derBytes, err := PrivateKeyToBytes(priv)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(derBytes), nil
}

// Helper function: HexStringToPrivateKey (Hex then DER)
func HexStringToPrivateKey(hexString string) (*ecdsa.PrivateKey, error) {
	derBytes, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string to bytes: %w", err)
	}
	return BytesToPrivateKey(derBytes)
}

// Helper function: PublicKeyToHexString (DER then Hex)
func PublicKeyToHexString(pub *ecdsa.PublicKey) (string, error) {
	derBytes, err := PublicKeyToBytes(pub)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(derBytes), nil
}

// Helper function: HexStringToPublicKey (Hex then DER)
func HexStringToPublicKey(hexString string) (*ecdsa.PublicKey, error) {
	derBytes, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string to bytes: %w", err)
	}
	return BytesToPublicKey(derBytes)
}
