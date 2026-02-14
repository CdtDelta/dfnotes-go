package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const recoveryCodeCount = 10
const recoveryCodeBytes = 4

func GenerateTOTPSecret(issuer, accountName string) (*otp.Key, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		return nil, fmt.Errorf("generate TOTP: %w", err)
	}
	return key, nil
}

func ValidateTOTPCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

func GenerateRecoveryCodes() ([]string, error) {
	codes := make([]string, recoveryCodeCount)
	for i := range codes {
		b := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("generate recovery code: %w", err)
		}
		codes[i] = hex.EncodeToString(b)
	}
	return codes, nil
}

func EncryptTOTPSecret(key []byte, secret string) ([]byte, error) {
	return Encrypt(key, []byte(secret))
}

func DecryptTOTPSecret(key, encryptedSecret []byte) (string, error) {
	plaintext, err := Decrypt(key, encryptedSecret)
	if err != nil {
		return "", fmt.Errorf("decrypt TOTP secret: %w", err)
	}
	return string(plaintext), nil
}

func ValidateRecoveryCode(code string, hashedCodes []string) (int, bool) {
	hashed := HashRecoveryCode(code)
	for i, h := range hashedCodes {
		if h == hashed {
			return i, true
		}
	}
	return -1, false
}

func GenerateTOTPCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}
