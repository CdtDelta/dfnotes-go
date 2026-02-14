package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"dfnotes-go/internal/models"
)

type UserRepo struct {
	db *DB
}

func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *models.UserIdentity) error {
	var recoveryCodes *string
	if len(user.RecoveryCodesHash) > 0 {
		data, _ := json.Marshal(user.RecoveryCodesHash)
		s := string(data)
		recoveryCodes = &s
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (user_id, name, organization, public_key, encrypted_private_key, salt, totp_enabled, totp_secret_encrypted, recovery_codes_hash, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.UserID, user.Name, user.Organization, user.PublicKey, user.EncryptedPrivateKey,
		user.Salt, user.TOTPEnabled, nullBytes(user.TOTPSecretEncrypted),
		recoveryCodes, FormatTime(user.CreatedAt),
	)
	return wrapError(err)
}

func (r *UserRepo) GetByID(ctx context.Context, userID string) (*models.UserIdentity, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT user_id, name, organization, public_key, encrypted_private_key, salt, totp_enabled, totp_secret_encrypted, recovery_codes_hash, created_at
		 FROM users WHERE user_id = ?`, userID)

	var u models.UserIdentity
	var createdAt string
	var totpSecret []byte
	var recoveryCodes sql.NullString

	err := row.Scan(&u.UserID, &u.Name, &u.Organization, &u.PublicKey, &u.EncryptedPrivateKey,
		&u.Salt, &u.TOTPEnabled, &totpSecret, &recoveryCodes, &createdAt)
	if err != nil {
		return nil, wrapError(err)
	}

	u.TOTPSecretEncrypted = totpSecret
	if recoveryCodes.Valid {
		json.Unmarshal([]byte(recoveryCodes.String), &u.RecoveryCodesHash)
	}

	u.CreatedAt, _ = ParseTime(createdAt)
	return &u, nil
}

func (r *UserRepo) GetFirst(ctx context.Context) (*models.UserIdentity, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT user_id, name, organization, public_key, encrypted_private_key, salt, totp_enabled, totp_secret_encrypted, recovery_codes_hash, created_at
		 FROM users LIMIT 1`)

	var u models.UserIdentity
	var createdAt string
	var totpSecret []byte
	var recoveryCodes sql.NullString

	err := row.Scan(&u.UserID, &u.Name, &u.Organization, &u.PublicKey, &u.EncryptedPrivateKey,
		&u.Salt, &u.TOTPEnabled, &totpSecret, &recoveryCodes, &createdAt)
	if err != nil {
		return nil, wrapError(err)
	}

	u.TOTPSecretEncrypted = totpSecret
	if recoveryCodes.Valid {
		json.Unmarshal([]byte(recoveryCodes.String), &u.RecoveryCodesHash)
	}

	u.CreatedAt, _ = ParseTime(createdAt)
	return &u, nil
}

func (r *UserRepo) Update(ctx context.Context, user *models.UserIdentity) error {
	var recoveryCodes *string
	if len(user.RecoveryCodesHash) > 0 {
		data, _ := json.Marshal(user.RecoveryCodesHash)
		s := string(data)
		recoveryCodes = &s
	}

	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET name=?, organization=?, public_key=?, encrypted_private_key=?, salt=?, totp_enabled=?, totp_secret_encrypted=?, recovery_codes_hash=?
		 WHERE user_id=?`,
		user.Name, user.Organization, user.PublicKey, user.EncryptedPrivateKey,
		user.Salt, user.TOTPEnabled, nullBytes(user.TOTPSecretEncrypted),
		recoveryCodes, user.UserID,
	)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func nullBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

func wrapError(err error) error {
	if err == nil {
		return nil
	}
	if err == sql.ErrNoRows {
		return models.ErrNotFound
	}
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed") {
		return models.ErrDuplicateKey
	}
	if strings.Contains(msg, "FOREIGN KEY constraint failed") {
		return models.ErrIntegrityViolation
	}
	return err
}

func checkRowsAffected(result sql.Result) error {
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}
