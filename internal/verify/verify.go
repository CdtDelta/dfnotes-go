package verify

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"dfnotes-go/internal/crypto"
	"dfnotes-go/internal/models"
)

// Verdict is the per-block outcome of chain verification.
type Verdict string

const (
	VerdictVerified   Verdict = "verified"
	VerdictTampered   Verdict = "tampered"
	VerdictChainBreak Verdict = "chain_break"
)

// BlockCheck holds the raw per-block check outcomes.
type BlockCheck struct {
	Decrypted      bool // ciphertext decrypted (GCM auth tag ok)
	HashChecked    bool // false when the body did not decrypt
	HashValid      bool // recomputed content hash == stored content hash
	SignatureValid bool // signature over the canonical payload verifies
}

// BlockResult is the per-block verdict for the chain.
type BlockResult struct {
	Sequence       int    // 1-based position in chain order
	BlockID        string
	EvidenceItemID string // "" for master notes; raw id otherwise (service maps to item_number)
	CommittedAt    string // RFC3339 UTC
	IsAmendment    bool
	IsGenesis      bool
	Decrypted      bool
	HashChecked    bool
	HashValid      bool
	SignatureValid bool
	LinkValid      bool // genesis: true (n/a). otherwise prev_hash == predecessor content hash
	Verdict        Verdict
	Detail         string
}

// Result is the whole-chain outcome.
type Result struct {
	ChainIntact     bool
	TotalBlocks     int
	FailedBlocks    int
	FirstFailureSeq int // 0 if none
	Blocks          []BlockResult
}

// BlockIntegrity verifies a single block's own integrity: decrypt, recompute
// hash, verify signature. It does NOT check chain linkage. Returns the checks and
// the decrypted plaintext (nil if decrypt failed) so callers can reuse the
// plaintext without decrypting twice. Used by both Chain (below) and the live note
// list path so the badge and the tool run byte identical per-block logic.
func BlockIntegrity(b models.NoteBlock, pubKey ed25519.PublicKey, decrypt func([]byte) ([]byte, error)) (BlockCheck, []byte) {
	var c BlockCheck

	// Signature and hash-of-record both verify against stored fields and do not
	// need the plaintext, so evaluate the signature even if decryption fails.
	payload := crypto.SigningPayload(b.ContentHash, b.PrevHash, b.CreatedAt, b.BlockID)
	c.SignatureValid = crypto.Verify(pubKey, payload, b.Signature)

	plaintext, err := decrypt(b.EncryptedBody)
	if err != nil {
		c.Decrypted = false
		c.HashChecked = false
		c.HashValid = false
		return c, nil
	}
	c.Decrypted = true
	c.HashChecked = true
	c.HashValid = crypto.HashContent(plaintext) == b.ContentHash
	return c, plaintext
}

// Chain verifies a full case chain. blocks MUST be supplied in chain (insertion)
// order, oldest first. pubKey is the committing identity's public key. decrypt
// decrypts a ciphertext body with the active case key.
func Chain(blocks []models.NoteBlock, pubKey ed25519.PublicKey, decrypt func([]byte) ([]byte, error)) Result {
	res := Result{TotalBlocks: len(blocks), ChainIntact: true}

	for i := range blocks {
		b := blocks[i]
		check, _ := BlockIntegrity(b, pubKey, decrypt)

		br := BlockResult{
			Sequence:       i + 1,
			BlockID:        b.BlockID,
			CommittedAt:    b.CreatedAt.UTC().Truncate(time.Second).Format(time.RFC3339),
			IsAmendment:    b.AmendsBlockID != nil,
			Decrypted:      check.Decrypted,
			HashChecked:    check.HashChecked,
			HashValid:      check.HashValid,
			SignatureValid: check.SignatureValid,
		}
		if b.EvidenceItemID != nil {
			br.EvidenceItemID = *b.EvidenceItemID
		}

		// Linkage. The first positional block is genesis iff its prev_hash is the
		// literal "genesis". Genesis has no predecessor, so its link is n/a (true).
		if i == 0 {
			if b.PrevHash == "genesis" {
				br.IsGenesis = true
				br.LinkValid = true
			} else {
				// Real genesis is missing or its prev_hash was altered (the latter
				// would also fail the signature, handled by precedence below).
				br.LinkValid = false
			}
		} else {
			br.LinkValid = b.PrevHash == blocks[i-1].ContentHash
		}

		// Verdict precedence: own-block tampering outranks a downstream link break,
		// so the root cause is named on the block that actually changed.
		switch {
		case !check.Decrypted:
			br.Verdict = VerdictTampered
			br.Detail = "decryption failed (ciphertext altered)"
		case !check.HashValid:
			br.Verdict = VerdictTampered
			br.Detail = "content hash mismatch"
		case !check.SignatureValid:
			br.Verdict = VerdictTampered
			br.Detail = "signature invalid (prev-hash, timestamp, id, or signature altered)"
		case !br.LinkValid:
			br.Verdict = VerdictChainBreak
			if i == 0 {
				br.Detail = "expected genesis block, predecessor missing"
			} else if res.Blocks[i-1].Verdict == VerdictTampered {
				br.Detail = fmt.Sprintf("chains to altered block %d", i)
			} else {
				br.Detail = "does not chain to prior block (deleted or reordered)"
			}
		default:
			br.Verdict = VerdictVerified
		}

		if br.Verdict != VerdictVerified {
			res.ChainIntact = false
			res.FailedBlocks++
			if res.FirstFailureSeq == 0 {
				res.FirstFailureSeq = br.Sequence
			}
		}
		res.Blocks = append(res.Blocks, br)
	}

	return res
}
