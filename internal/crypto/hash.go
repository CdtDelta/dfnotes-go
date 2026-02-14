package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashContent(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func HashRecoveryCode(code string) string {
	return HashContent([]byte(code))
}
