package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"dfnotes-go/internal/crypto"
	"dfnotes-go/internal/models"

	"github.com/google/uuid"
)

type SetupRequest struct {
	Name         string `json:"name"`
	Organization string `json:"organization"`
	Password     string `json:"password"`
	EnableTOTP   bool   `json:"enable_totp"`
}

type SetupResponse struct {
	UserID        string   `json:"user_id"`
	TOTPEnabled   bool     `json:"totp_enabled"`
	TOTPURL       string   `json:"totp_url,omitempty"`
	RecoveryCodes []string `json:"recovery_codes,omitempty"`
}

type LoginRequest struct {
	Password string `json:"password"`
	TOTPCode string `json:"totp_code,omitempty"`
}

type LoginResponse struct {
	UserID       string `json:"user_id"`
	Name         string `json:"name"`
	Organization string `json:"organization"`
	TOTPEnabled  bool   `json:"totp_enabled"`
}

type LoginScreenInfo struct {
	UserID       string `json:"user_id"`
	Name         string `json:"name"`
	Organization string `json:"organization"`
	TOTPEnabled  bool   `json:"totp_enabled"`
}

type IdentityService struct {
	userRepo  models.UserRepository
	auditRepo models.AuditLogRepository
	session   *Session
}

func NewIdentityService(userRepo models.UserRepository, auditRepo models.AuditLogRepository, session *Session) *IdentityService {
	return &IdentityService{
		userRepo:  userRepo,
		auditRepo: auditRepo,
		session:   session,
	}
}

func (s *IdentityService) GetFirstUser(ctx context.Context) (*models.UserIdentity, error) {
	return s.userRepo.GetFirst(ctx)
}

func (s *IdentityService) GetLoginScreenInfo(ctx context.Context) (*LoginScreenInfo, error) {
	user, err := s.userRepo.GetFirst(ctx)
	if err != nil {
		return nil, err
	}
	return &LoginScreenInfo{
		UserID:       user.UserID,
		Name:         user.Name,
		Organization: user.Organization,
		TOTPEnabled:  user.TOTPEnabled,
	}, nil
}

func (s *IdentityService) Setup(ctx context.Context, req SetupRequest) (*SetupResponse, error) {
	if req.Name == "" || req.Password == "" {
		return nil, errors.New("name and password are required")
	}

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return nil, err
	}

	derivedKey := crypto.DeriveKey(req.Password, salt)

	pubKey, privKey, err := crypto.GenerateSigningKeyPair()
	if err != nil {
		return nil, err
	}

	encPrivKey, err := crypto.EncryptPrivateKey(derivedKey, privKey)
	if err != nil {
		return nil, err
	}

	userID := uuid.New().String()
	now := time.Now().UTC()

	user := &models.UserIdentity{
		UserID:              userID,
		Name:                req.Name,
		Organization:        req.Organization,
		PublicKey:           pubKey,
		EncryptedPrivateKey: encPrivKey,
		Salt:                salt,
		TOTPEnabled:         false,
		CreatedAt:           now,
	}

	resp := &SetupResponse{
		UserID:      userID,
		TOTPEnabled: false,
	}

	if req.EnableTOTP {
		totpKey, err := crypto.GenerateTOTPSecret("dfnotes-go", req.Name)
		if err != nil {
			return nil, err
		}

		encTOTP, err := crypto.EncryptTOTPSecret(derivedKey, totpKey.Secret())
		if err != nil {
			return nil, err
		}

		recoveryCodes, err := crypto.GenerateRecoveryCodes()
		if err != nil {
			return nil, err
		}

		hashedCodes := make([]string, len(recoveryCodes))
		for i, code := range recoveryCodes {
			hashedCodes[i] = crypto.HashRecoveryCode(code)
		}

		user.TOTPEnabled = true
		user.TOTPSecretEncrypted = encTOTP
		user.RecoveryCodesHash = hashedCodes

		resp.TOTPEnabled = true
		resp.TOTPURL = totpKey.URL()
		resp.RecoveryCodes = recoveryCodes
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	details, _ := json.Marshal(map[string]string{"action": "identity_setup", "user_id": userID})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		UserID:     userID,
		Action:     models.AuditActionCreate,
		EntityType: "user",
		EntityID:   userID,
		Details:    details,
		CreatedAt:  now,
	})

	s.session.SetAuthenticated(user, derivedKey, privKey)

	return resp, nil
}

func (s *IdentityService) ConfirmTOTPSetup(ctx context.Context, code string) (bool, error) {
	if !s.session.IsAuthenticated() {
		return false, errors.New("not authenticated")
	}

	user := s.session.User()
	if !user.TOTPEnabled || len(user.TOTPSecretEncrypted) == 0 {
		return false, errors.New("TOTP not enabled")
	}

	secret, err := crypto.DecryptTOTPSecret(s.session.DerivedKey(), user.TOTPSecretEncrypted)
	if err != nil {
		return false, err
	}

	if !crypto.ValidateTOTPCode(secret, code) {
		return false, models.ErrTOTPInvalid
	}

	return true, nil
}

func (s *IdentityService) LoginFirstUser(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.userRepo.GetFirst(ctx)
	if err != nil {
		return nil, err
	}

	derivedKey := crypto.DeriveKey(req.Password, user.Salt)

	privKey, err := crypto.DecryptPrivateKey(derivedKey, user.EncryptedPrivateKey)
	if err != nil {
		return nil, models.ErrInvalidPassword
	}

	if user.TOTPEnabled {
		if req.TOTPCode == "" {
			return nil, models.ErrTOTPRequired
		}

		secret, err := crypto.DecryptTOTPSecret(derivedKey, user.TOTPSecretEncrypted)
		if err != nil {
			return nil, models.ErrInvalidPassword
		}

		if !crypto.ValidateTOTPCode(secret, req.TOTPCode) {
			// Try recovery codes
			idx, valid := crypto.ValidateRecoveryCode(req.TOTPCode, user.RecoveryCodesHash)
			if !valid {
				return nil, models.ErrTOTPInvalid
			}
			// Mark recovery code as used
			user.RecoveryCodesHash[idx] = "used"
			s.userRepo.Update(ctx, user)
		}
	}

	s.session.SetAuthenticated(user, derivedKey, privKey)

	now := time.Now().UTC()
	details, _ := json.Marshal(map[string]string{"action": "login"})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		UserID:     user.UserID,
		Action:     models.AuditActionLogin,
		EntityType: "user",
		EntityID:   user.UserID,
		Details:    details,
		CreatedAt:  now,
	})

	return &LoginResponse{
		UserID:       user.UserID,
		Name:         user.Name,
		Organization: user.Organization,
		TOTPEnabled:  user.TOTPEnabled,
	}, nil
}
