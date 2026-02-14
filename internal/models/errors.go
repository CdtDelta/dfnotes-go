package models

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrDuplicateKey       = errors.New("duplicate key")
	ErrIntegrityViolation = errors.New("integrity violation")
	ErrDecryptionFailed   = errors.New("decryption failed")
	ErrInvalidSignature   = errors.New("invalid signature")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrChainBroken        = errors.New("hash chain broken")
	ErrTOTPRequired       = errors.New("TOTP required")
	ErrTOTPInvalid        = errors.New("TOTP code invalid")
	ErrRecoveryCodeUsed   = errors.New("recovery code already used")
)
