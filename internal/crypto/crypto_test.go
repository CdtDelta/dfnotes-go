package crypto

import (
	"bytes"
	"crypto/ed25519"
	"testing"
)

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt: %v", err)
	}
	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt: %v", err)
	}
	if len(salt1) != saltLen {
		t.Fatalf("expected salt length %d, got %d", saltLen, len(salt1))
	}
	if bytes.Equal(salt1, salt2) {
		t.Fatal("two salts should not be equal")
	}
}

func TestDeriveKey(t *testing.T) {
	salt, _ := GenerateSalt()
	key1 := DeriveKey("password", salt)
	key2 := DeriveKey("password", salt)
	if !bytes.Equal(key1, key2) {
		t.Fatal("same password+salt should produce same key")
	}
	if len(key1) != argonKeyLen {
		t.Fatalf("expected key length %d, got %d", argonKeyLen, len(key1))
	}

	key3 := DeriveKey("different", salt)
	if bytes.Equal(key1, key3) {
		t.Fatal("different passwords should produce different keys")
	}
}

func TestHashContent(t *testing.T) {
	h1 := HashContent([]byte("hello"))
	h2 := HashContent([]byte("hello"))
	if h1 != h2 {
		t.Fatal("same input should produce same hash")
	}
	h3 := HashContent([]byte("world"))
	if h1 == h3 {
		t.Fatal("different inputs should produce different hashes")
	}
	if len(h1) != 64 {
		t.Fatalf("expected SHA-256 hex length 64, got %d", len(h1))
	}
}

func TestHashRecoveryCode(t *testing.T) {
	h := HashRecoveryCode("abc123")
	if h != HashContent([]byte("abc123")) {
		t.Fatal("HashRecoveryCode should delegate to HashContent")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := DeriveKey("test-password", make([]byte, saltLen))
	plaintext := []byte("sensitive data for testing encryption")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("decrypted data should match original plaintext")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := DeriveKey("password1", make([]byte, saltLen))
	key2 := DeriveKey("password2", make([]byte, saltLen))
	plaintext := []byte("secret")

	ciphertext, _ := Encrypt(key1, plaintext)
	_, err := Decrypt(key2, ciphertext)
	if err == nil {
		t.Fatal("decrypting with wrong key should fail")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := DeriveKey("test", make([]byte, saltLen))
	_, err := Decrypt(key, []byte("short"))
	if err == nil {
		t.Fatal("decrypting too-short ciphertext should fail")
	}
}

func TestSigningRoundTrip(t *testing.T) {
	pub, priv, err := GenerateSigningKeyPair()
	if err != nil {
		t.Fatalf("GenerateSigningKeyPair: %v", err)
	}

	message := []byte("test message to sign")
	sig := Sign(priv, message)

	if !Verify(pub, message, sig) {
		t.Fatal("valid signature should verify")
	}
	if Verify(pub, []byte("tampered"), sig) {
		t.Fatal("tampered message should not verify")
	}
}

func TestEncryptDecryptPrivateKey(t *testing.T) {
	_, priv, _ := GenerateSigningKeyPair()
	key := DeriveKey("password", make([]byte, saltLen))

	encrypted, err := EncryptPrivateKey(key, priv)
	if err != nil {
		t.Fatalf("EncryptPrivateKey: %v", err)
	}

	decrypted, err := DecryptPrivateKey(key, encrypted)
	if err != nil {
		t.Fatalf("DecryptPrivateKey: %v", err)
	}
	if !bytes.Equal(decrypted, priv) {
		t.Fatal("decrypted private key should match original")
	}

	// Verify decrypted key is functionally valid
	message := []byte("verify key works")
	sig := ed25519.Sign(decrypted, message)
	pub := decrypted.Public().(ed25519.PublicKey)
	if !ed25519.Verify(pub, message, sig) {
		t.Fatal("decrypted key should produce valid signatures")
	}
}

func TestTOTPSecretGeneration(t *testing.T) {
	key, err := GenerateTOTPSecret("dfnotes", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret: %v", err)
	}
	if key.Secret() == "" {
		t.Fatal("TOTP secret should not be empty")
	}
}

func TestTOTPValidation(t *testing.T) {
	key, _ := GenerateTOTPSecret("dfnotes", "test@example.com")
	code, err := GenerateTOTPCode(key.Secret())
	if err != nil {
		t.Fatalf("GenerateTOTPCode: %v", err)
	}
	if !ValidateTOTPCode(key.Secret(), code) {
		t.Fatal("generated code should validate")
	}
	if ValidateTOTPCode(key.Secret(), "000000") {
		// This could theoretically pass but is extremely unlikely
		t.Log("warning: 000000 validated (unlikely but possible)")
	}
}

func TestEncryptDecryptTOTPSecret(t *testing.T) {
	key := DeriveKey("password", make([]byte, saltLen))
	secret := "JBSWY3DPEHPK3PXP"

	encrypted, err := EncryptTOTPSecret(key, secret)
	if err != nil {
		t.Fatalf("EncryptTOTPSecret: %v", err)
	}

	decrypted, err := DecryptTOTPSecret(key, encrypted)
	if err != nil {
		t.Fatalf("DecryptTOTPSecret: %v", err)
	}
	if decrypted != secret {
		t.Fatalf("expected %q, got %q", secret, decrypted)
	}
}

func TestGenerateRecoveryCodes(t *testing.T) {
	codes, err := GenerateRecoveryCodes()
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes: %v", err)
	}
	if len(codes) != recoveryCodeCount {
		t.Fatalf("expected %d codes, got %d", recoveryCodeCount, len(codes))
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		if len(code) != recoveryCodeBytes*2 {
			t.Fatalf("expected code length %d, got %d", recoveryCodeBytes*2, len(code))
		}
		if seen[code] {
			t.Fatalf("duplicate recovery code: %s", code)
		}
		seen[code] = true
	}
}

func TestValidateRecoveryCode(t *testing.T) {
	codes, _ := GenerateRecoveryCodes()
	hashed := make([]string, len(codes))
	for i, c := range codes {
		hashed[i] = HashRecoveryCode(c)
	}

	idx, ok := ValidateRecoveryCode(codes[3], hashed)
	if !ok || idx != 3 {
		t.Fatalf("expected index 3, got %d (ok=%v)", idx, ok)
	}

	idx, ok = ValidateRecoveryCode("invalid-code", hashed)
	if ok || idx != -1 {
		t.Fatalf("invalid code should not match (idx=%d, ok=%v)", idx, ok)
	}
}
