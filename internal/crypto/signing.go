package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"time"
)

// payloadSep is ASCII Unit Separator (0x1F). It cannot appear in hex hashes,
// RFC3339 timestamps, or UUIDs, so it is an unambiguous field delimiter.
const payloadSep = "\x1f"

// SigningPayload builds the canonical byte sequence signed for a note block.
// created_at is canonicalized to UTC truncated to whole seconds, so that the
// commit-time value and the value re-read from the database produce a byte
// identical payload even if the stored precision differs. The truncation is the
// safeguard: do not change it without also proving the DB round-trip stays lossless.
func SigningPayload(contentHash, prevHash string, createdAt time.Time, blockID string) []byte {
	ts := createdAt.UTC().Truncate(time.Second).Format(time.RFC3339)
	return []byte(contentHash + payloadSep + prevHash + payloadSep + ts + payloadSep + blockID)
}

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
