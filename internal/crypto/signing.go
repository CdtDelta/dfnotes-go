package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
)

func GenerateSigningKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key pair: %w", err)
	}
	return pub, priv, nil
}

func Sign(privateKey ed25519.PrivateKey, message []byte) []byte {
	return ed25519.Sign(privateKey, message)
}

func Verify(publicKey ed25519.PublicKey, message, sig []byte) bool {
	return ed25519.Verify(publicKey, message, sig)
}

func EncryptPrivateKey(key []byte, privateKey ed25519.PrivateKey) ([]byte, error) {
	return Encrypt(key, privateKey)
}

func DecryptPrivateKey(key, encryptedKey []byte) (ed25519.PrivateKey, error) {
	plaintext, err := Decrypt(key, encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt private key: %w", err)
	}
	return ed25519.PrivateKey(plaintext), nil
}
