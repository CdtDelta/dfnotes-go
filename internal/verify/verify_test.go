package verify

import (
	"crypto/ed25519"
	"testing"
	"time"

	"dfnotes-go/internal/crypto"
	"dfnotes-go/internal/models"
)

// testKeys generates a fresh ed25519 key pair for tests.
func testKeys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := crypto.GenerateSigningKeyPair()
	if err != nil {
		t.Fatalf("GenerateSigningKeyPair: %v", err)
	}
	return pub, priv
}

// testCaseKey generates a 32-byte AES key for tests.
func testCaseKey() []byte {
	salt := make([]byte, 16)
	return crypto.DeriveKey("test-case-password", salt)
}

// makeBlock constructs a NoteBlock with a valid signature over the canonical payload.
func makeBlock(t *testing.T, priv ed25519.PrivateKey, caseKey []byte, blockID, caseID, content, prevHash string, createdAt time.Time) models.NoteBlock {
	t.Helper()
	contentHash := crypto.HashContent([]byte(content))
	encrypted, err := crypto.Encrypt(caseKey, []byte(content))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	sig := crypto.Sign(priv, crypto.SigningPayload(contentHash, prevHash, createdAt, blockID))
	return models.NoteBlock{
		BlockID:       blockID,
		CaseID:        caseID,
		ContentHash:   contentHash,
		PrevHash:      prevHash,
		Signature:     sig,
		EncryptedBody: encrypted,
		CreatedAt:     createdAt,
	}
}

func decryptFunc(caseKey []byte) func([]byte) ([]byte, error) {
	return func(ct []byte) ([]byte, error) {
		return crypto.Decrypt(caseKey, ct)
	}
}

func TestChainEmpty(t *testing.T) {
	pub, _ := testKeys(t)
	res := Chain(nil, pub, func([]byte) ([]byte, error) { return nil, nil })
	if !res.ChainIntact {
		t.Error("empty chain should be intact")
	}
	if res.TotalBlocks != 0 {
		t.Errorf("expected 0 blocks, got %d", res.TotalBlocks)
	}
	if len(res.Blocks) != 0 {
		t.Errorf("expected no block results, got %d", len(res.Blocks))
	}
}

func TestChainSingleGenesis(t *testing.T) {
	pub, priv := testKeys(t)
	key := testCaseKey()
	now := time.Now().UTC().Truncate(time.Second)

	b := makeBlock(t, priv, key, "b1", "case1", "hello world", "genesis", now)
	res := Chain([]models.NoteBlock{b}, pub, decryptFunc(key))

	if !res.ChainIntact {
		t.Error("single genesis block should be intact")
	}
	if res.TotalBlocks != 1 || len(res.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", res.TotalBlocks)
	}
	br := res.Blocks[0]
	if br.Verdict != VerdictVerified {
		t.Errorf("expected verified, got %s (%s)", br.Verdict, br.Detail)
	}
	if !br.IsGenesis {
		t.Error("block should be marked genesis")
	}
	if !br.LinkValid {
		t.Error("genesis link should be valid (n/a)")
	}
}

func TestChainClean3Blocks(t *testing.T) {
	pub, priv := testKeys(t)
	key := testCaseKey()
	now := time.Now().UTC().Truncate(time.Second)

	b1 := makeBlock(t, priv, key, "b1", "c1", "block one", "genesis", now)
	b2 := makeBlock(t, priv, key, "b2", "c1", "block two", b1.ContentHash, now.Add(time.Second))
	b3 := makeBlock(t, priv, key, "b3", "c1", "block three", b2.ContentHash, now.Add(2*time.Second))

	res := Chain([]models.NoteBlock{b1, b2, b3}, pub, decryptFunc(key))

	if !res.ChainIntact {
		t.Errorf("clean chain should be intact; first failure seq %d", res.FirstFailureSeq)
	}
	if res.FailedBlocks != 0 {
		t.Errorf("expected 0 failed blocks, got %d", res.FailedBlocks)
	}
	for i, br := range res.Blocks {
		if br.Verdict != VerdictVerified {
			t.Errorf("block %d: expected verified, got %s (%s)", i+1, br.Verdict, br.Detail)
		}
	}
}

func TestChainCiphertextFlipBlock2(t *testing.T) {
	pub, priv := testKeys(t)
	key := testCaseKey()
	now := time.Now().UTC().Truncate(time.Second)

	b1 := makeBlock(t, priv, key, "b1", "c1", "block one", "genesis", now)
	b2 := makeBlock(t, priv, key, "b2", "c1", "block two", b1.ContentHash, now.Add(time.Second))
	b3 := makeBlock(t, priv, key, "b3", "c1", "block three", b2.ContentHash, now.Add(2*time.Second))

	// Flip a byte in block 2's ciphertext (past the GCM nonce prefix).
	tampered := make([]byte, len(b2.EncryptedBody))
	copy(tampered, b2.EncryptedBody)
	tampered[20] ^= 0xFF
	b2.EncryptedBody = tampered

	res := Chain([]models.NoteBlock{b1, b2, b3}, pub, decryptFunc(key))

	if res.ChainIntact {
		t.Error("chain with ciphertext flip should not be intact")
	}
	if res.Blocks[0].Verdict != VerdictVerified {
		t.Errorf("block 1 should be verified, got %s", res.Blocks[0].Verdict)
	}
	if res.Blocks[1].Verdict != VerdictTampered {
		t.Errorf("block 2 should be tampered, got %s (%s)", res.Blocks[1].Verdict, res.Blocks[1].Detail)
	}
	if res.Blocks[2].Verdict != VerdictVerified {
		t.Errorf("block 3 should be verified (sig and chain still ok), got %s", res.Blocks[2].Verdict)
	}
}

func TestChainContentHashEditBlock2(t *testing.T) {
	pub, priv := testKeys(t)
	key := testCaseKey()
	now := time.Now().UTC().Truncate(time.Second)

	b1 := makeBlock(t, priv, key, "b1", "c1", "block one", "genesis", now)
	b2 := makeBlock(t, priv, key, "b2", "c1", "block two", b1.ContentHash, now.Add(time.Second))
	b3 := makeBlock(t, priv, key, "b3", "c1", "block three", b2.ContentHash, now.Add(2*time.Second))

	// Silently alter block 2's stored content_hash field (leaving encrypted body and sig intact).
	orig2Hash := b2.ContentHash
	b2.ContentHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	// b3 was built pointing at orig2Hash, not the altered one, so b3.PrevHash != b2.ContentHash.
	_ = orig2Hash

	res := Chain([]models.NoteBlock{b1, b2, b3}, pub, decryptFunc(key))

	if res.ChainIntact {
		t.Error("chain with hash edit should not be intact")
	}
	if res.Blocks[0].Verdict != VerdictVerified {
		t.Errorf("block 1: expected verified, got %s", res.Blocks[0].Verdict)
	}
	// Block 2: hash mismatch (recomputed != stored altered value).
	if res.Blocks[1].Verdict != VerdictTampered {
		t.Errorf("block 2: expected tampered, got %s (%s)", res.Blocks[1].Verdict, res.Blocks[1].Detail)
	}
	// Block 3: prev_hash == orig2Hash, but b2.ContentHash is now the altered value, so chain break.
	if res.Blocks[2].Verdict != VerdictChainBreak {
		t.Errorf("block 3: expected chain_break, got %s (%s)", res.Blocks[2].Verdict, res.Blocks[2].Detail)
	}
}

func TestChainPrevHashEditBlock2(t *testing.T) {
	pub, priv := testKeys(t)
	key := testCaseKey()
	now := time.Now().UTC().Truncate(time.Second)

	b1 := makeBlock(t, priv, key, "b1", "c1", "block one", "genesis", now)
	b2 := makeBlock(t, priv, key, "b2", "c1", "block two", b1.ContentHash, now.Add(time.Second))
	b3 := makeBlock(t, priv, key, "b3", "c1", "block three", b2.ContentHash, now.Add(2*time.Second))

	// Alter block 2's prev_hash field (breaks signature, self-contained).
	b2.PrevHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	res := Chain([]models.NoteBlock{b1, b2, b3}, pub, decryptFunc(key))

	if res.ChainIntact {
		t.Error("chain should not be intact")
	}
	// Block 2: signature now invalid (prev_hash is part of payload).
	if res.Blocks[1].Verdict != VerdictTampered {
		t.Errorf("block 2: expected tampered (sig invalid), got %s (%s)", res.Blocks[1].Verdict, res.Blocks[1].Detail)
	}
	// Block 3: its prev_hash still points at b2.ContentHash which is unchanged, so chain is valid.
	// Block 2's sig is invalid but content_hash is unchanged, so block 3 chains correctly.
	if res.Blocks[2].Verdict != VerdictVerified {
		t.Errorf("block 3: expected verified, got %s (%s)", res.Blocks[2].Verdict, res.Blocks[2].Detail)
	}
}

func TestChainCreatedAtEditBlock2(t *testing.T) {
	pub, priv := testKeys(t)
	key := testCaseKey()
	now := time.Now().UTC().Truncate(time.Second)

	b1 := makeBlock(t, priv, key, "b1", "c1", "block one", "genesis", now)
	b2 := makeBlock(t, priv, key, "b2", "c1", "block two", b1.ContentHash, now.Add(time.Second))
	b3 := makeBlock(t, priv, key, "b3", "c1", "block three", b2.ContentHash, now.Add(2*time.Second))

	// Alter block 2's created_at (breaks signature).
	b2.CreatedAt = b2.CreatedAt.Add(24 * time.Hour)

	res := Chain([]models.NoteBlock{b1, b2, b3}, pub, decryptFunc(key))

	if res.ChainIntact {
		t.Error("chain should not be intact")
	}
	if res.Blocks[1].Verdict != VerdictTampered {
		t.Errorf("block 2: expected tampered (sig invalid), got %s (%s)", res.Blocks[1].Verdict, res.Blocks[1].Detail)
	}
}

func TestChainBlockIDEditBlock2(t *testing.T) {
	pub, priv := testKeys(t)
	key := testCaseKey()
	now := time.Now().UTC().Truncate(time.Second)

	b1 := makeBlock(t, priv, key, "b1", "c1", "block one", "genesis", now)
	b2 := makeBlock(t, priv, key, "b2", "c1", "block two", b1.ContentHash, now.Add(time.Second))
	b3 := makeBlock(t, priv, key, "b3", "c1", "block three", b2.ContentHash, now.Add(2*time.Second))

	// Alter block 2's block_id (breaks signature).
	b2.BlockID = "altered-block-id"

	res := Chain([]models.NoteBlock{b1, b2, b3}, pub, decryptFunc(key))

	if res.ChainIntact {
		t.Error("chain should not be intact")
	}
	if res.Blocks[1].Verdict != VerdictTampered {
		t.Errorf("block 2: expected tampered (sig invalid), got %s (%s)", res.Blocks[1].Verdict, res.Blocks[1].Detail)
	}
}

func TestChainDeletedMiddleBlock(t *testing.T) {
	pub, priv := testKeys(t)
	key := testCaseKey()
	now := time.Now().UTC().Truncate(time.Second)

	b1 := makeBlock(t, priv, key, "b1", "c1", "block one", "genesis", now)
	b2 := makeBlock(t, priv, key, "b2", "c1", "block two", b1.ContentHash, now.Add(time.Second))
	b3 := makeBlock(t, priv, key, "b3", "c1", "block three", b2.ContentHash, now.Add(2*time.Second))
	_ = b2 // b2 deleted; pass [b1, b3]

	res := Chain([]models.NoteBlock{b1, b3}, pub, decryptFunc(key))

	if res.ChainIntact {
		t.Error("chain should not be intact after middle block deletion")
	}
	if res.Blocks[0].Verdict != VerdictVerified {
		t.Errorf("block 1: expected verified, got %s", res.Blocks[0].Verdict)
	}
	if res.Blocks[1].Verdict != VerdictChainBreak {
		t.Errorf("block 3 (now seq 2): expected chain_break, got %s (%s)", res.Blocks[1].Verdict, res.Blocks[1].Detail)
	}
}

func TestChainReorderedBlocks(t *testing.T) {
	pub, priv := testKeys(t)
	key := testCaseKey()
	now := time.Now().UTC().Truncate(time.Second)

	b1 := makeBlock(t, priv, key, "b1", "c1", "block one", "genesis", now)
	b2 := makeBlock(t, priv, key, "b2", "c1", "block two", b1.ContentHash, now.Add(time.Second))
	b3 := makeBlock(t, priv, key, "b3", "c1", "block three", b2.ContentHash, now.Add(2*time.Second))

	// Pass in reverse order: b3, b2, b1
	res := Chain([]models.NoteBlock{b3, b2, b1}, pub, decryptFunc(key))

	if res.ChainIntact {
		t.Error("reordered chain should not be intact")
	}
	// b3 is first but its prev_hash != "genesis", so it fails.
	if res.Blocks[0].Verdict == VerdictVerified {
		t.Errorf("reordered first block should not be verified, got %s", res.Blocks[0].Verdict)
	}
}

func TestChainWrongPublicKey(t *testing.T) {
	_, priv := testKeys(t)
	wrongPub, _ := testKeys(t)
	key := testCaseKey()
	now := time.Now().UTC().Truncate(time.Second)

	b1 := makeBlock(t, priv, key, "b1", "c1", "block one", "genesis", now)

	res := Chain([]models.NoteBlock{b1}, wrongPub, decryptFunc(key))

	if res.ChainIntact {
		t.Error("chain with wrong public key should not be intact")
	}
	if res.Blocks[0].Verdict != VerdictTampered {
		t.Errorf("expected tampered (sig invalid), got %s", res.Blocks[0].Verdict)
	}
}
