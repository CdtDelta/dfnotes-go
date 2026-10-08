package crypto

import (
	"crypto/rand"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 64 * 1024 // 64 MB
	argonIterations  = 3
	argonParallelism = 4
	argonKeyLen      = 32
	saltLen          = 16
)

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// GenerateCaseKey returns a random key for a case created without a case
// password. It is the same length as a DeriveKey output so both kinds of case
// key are interchangeable everywhere downstream.
func GenerateCaseKey() ([]byte, error) {
	key := make([]byte, argonKeyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLen)
}
