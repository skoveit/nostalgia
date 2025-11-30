package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Hardcoded keypair for demonstration (In production, use secure key management)
const (
	// This is a hardcoded private key (64 bytes in hex)
	HardcodedPrivateKeyHex = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"
	// Corresponding public key (32 bytes in hex)
	HardcodedPublicKeyHex = "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"
)

// GenerateKeypair generates a new Ed25519 keypair
func GenerateKeypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate keypair: %w", err)
	}
	return pub, priv, nil
}

// GetHardcodedPrivateKey returns the hardcoded private key
func GetHardcodedPrivateKey() (ed25519.PrivateKey, error) {
	privBytes, err := hex.DecodeString(HardcodedPrivateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}
	if len(privBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: got %d, want %d", len(privBytes), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(privBytes), nil
}

// GetHardcodedPublicKey returns the hardcoded public key
func GetHardcodedPublicKey() (ed25519.PublicKey, error) {
	pubBytes, err := hex.DecodeString(HardcodedPublicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: got %d, want %d", len(pubBytes), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(pubBytes), nil
}

// SignMessage signs a message using the provided private key
func SignMessage(message []byte, privateKey ed25519.PrivateKey) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size")
	}
	signature := ed25519.Sign(privateKey, message)
	return signature, nil
}

// VerifySignature verifies a signature using the provided public key
func VerifySignature(message []byte, signature []byte, publicKey ed25519.PublicKey) bool {
	if len(publicKey) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(publicKey, message, signature)
}

// SignCommand signs a command using the hardcoded private key
func SignCommand(cmd *Command) error {
	privateKey, err := GetHardcodedPrivateKey()
	if err != nil {
		return fmt.Errorf("failed to get private key: %w", err)
	}

	// Serialize command without signature
	cmd.Signature = nil
	data, err := cmd.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize command: %w", err)
	}

	// Sign the serialized data
	signature, err := SignMessage(data, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	cmd.Signature = signature
	return nil
}

// VerifyCommand verifies a command signature using the hardcoded public key
func VerifyCommand(cmd *Command) (bool, error) {
	publicKey, err := GetHardcodedPublicKey()
	if err != nil {
		return false, fmt.Errorf("failed to get public key: %w", err)
	}

	// Store signature and clear it for verification
	signature := cmd.Signature
	cmd.Signature = nil

	// Serialize command without signature
	data, err := cmd.Serialize()
	if err != nil {
		return false, fmt.Errorf("failed to serialize command: %w", err)
	}

	// Restore signature
	cmd.Signature = signature

	// Verify signature
	isValid := VerifySignature(data, signature, publicKey)
	return isValid, nil
}
