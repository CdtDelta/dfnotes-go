package services

import (
	"crypto/ed25519"
	"sync"

	"dfnotes-go/internal/models"
)

type Session struct {
	mu         sync.RWMutex
	user       *models.UserIdentity
	derivedKey []byte
	privateKey ed25519.PrivateKey
}

func NewSession() *Session {
	return &Session{}
}

func (s *Session) SetAuthenticated(user *models.UserIdentity, derivedKey []byte, privateKey ed25519.PrivateKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.user = user
	s.derivedKey = derivedKey
	s.privateKey = privateKey
}

func (s *Session) IsAuthenticated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.user != nil && s.derivedKey != nil
}

func (s *Session) User() *models.UserIdentity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.user
}

func (s *Session) DerivedKey() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.derivedKey
}

func (s *Session) PrivateKey() ed25519.PrivateKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.privateKey
}

func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Zero key material
	for i := range s.derivedKey {
		s.derivedKey[i] = 0
	}
	for i := range s.privateKey {
		s.privateKey[i] = 0
	}
	s.derivedKey = nil
	s.privateKey = nil
	s.user = nil
}
