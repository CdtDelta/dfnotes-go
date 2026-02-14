package models

import "time"

type UserIdentity struct {
	UserID              string    `json:"user_id"`
	Name                string    `json:"name"`
	Organization        string    `json:"organization"`
	PublicKey           []byte    `json:"public_key"`
	EncryptedPrivateKey []byte    `json:"encrypted_private_key"`
	Salt                []byte    `json:"salt"`
	TOTPEnabled         bool      `json:"totp_enabled"`
	TOTPSecretEncrypted []byte    `json:"totp_secret_encrypted,omitempty"`
	RecoveryCodesHash   []string  `json:"recovery_codes_hash,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}
