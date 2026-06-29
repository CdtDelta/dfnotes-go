# dfnotes-go: Chain Integrity Verification

This document describes the integrity model for dfnotes-go note blocks: what each
block carries, how verification works, what the verdicts mean, and how a third party
can independently re-verify an export without running dfnotes-go itself.

---

## What Each Committed Block Carries

When a note block is committed, dfnotes-go stores five things in the database for
each block:

**Encrypted content.** The block's markdown text, encrypted with AES-256-GCM using
the case-specific key. The key is derived from the case password via Argon2id and
stored wrapped by the application master key. The ciphertext includes a GCM
authentication tag that will fail to verify if the ciphertext bytes are altered.

**Content hash.** The SHA-256 hex digest of the plaintext content before encryption.
This is stored unencrypted. It is the fingerprint of the block's content.

**Previous-block hash.** The content hash of the immediately preceding committed
block in chain order (global per case, not per tab). The first block carries the
literal string "genesis". This forms the chain link.

**Ed25519 signature.** A digital signature over the canonical payload, produced with
the examiner's Ed25519 private key. The canonical payload is constructed from four
fields, joined by ASCII Unit Separator (0x1F, decimal 31) in this exact order:

    content_hash + 0x1F + previous_block_hash + 0x1F + committed_at + 0x1F + block_id

where `committed_at` is the RFC3339 UTC commit timestamp truncated to whole seconds.
The delimiter (0x1F) cannot appear in hex hashes, RFC3339 timestamps, or UUIDs, so
the field boundaries are unambiguous. The signature is stored as raw bytes in the
database and exported as base64 in chain_verification.json.

**Block metadata.** Block ID (UUID), case ID, evidence item association (null for
Master Notes blocks), commit timestamp, and examiner ID. Stored in plaintext.

---

## What Verification Checks, Per Block

The `internal/verify` package runs four checks for each block in chain order, oldest
first:

**1. Decrypt.** Attempt to decrypt the ciphertext body with AES-256-GCM. The GCM
authentication tag fails if the ciphertext was altered. A decryption failure makes
the hash check impossible (there is no plaintext to hash) and it is skipped.
The signature check is still attempted, because it operates on stored fields rather
than the plaintext.

**2. Hash.** Recompute SHA-256 of the decrypted plaintext and compare to the stored
content hash. A mismatch means either the plaintext content or the stored content
hash was altered after the block was committed.

**3. Signature.** Rebuild the canonical payload from the block's stored fields
(content hash, previous-block hash, committed_at, block_id) and verify the stored
Ed25519 signature against the examiner's public key. A failure means at least one of
those four fields was altered after the block was committed.

**4. Chain link.** For non-genesis blocks, confirm that the block's
previous_block_hash matches the content_hash of the immediately preceding block in
chain order. A mismatch means a block was deleted, reordered, or its predecessor's
content hash was altered.

---

## Verdict Taxonomy

Each block receives exactly one verdict.

**VERIFIED** -- all four checks passed. The block's content is unchanged, the chain
link is intact, and the signature is valid.

**TAMPERED** -- at least one per-block check failed. Three detail strings are
possible:

- "decryption failed (ciphertext altered)" -- GCM authentication tag did not verify;
  the encrypted bytes were changed.
- "content hash mismatch" -- the block decrypted successfully, but the SHA-256 of
  the plaintext does not match the stored content_hash; one of the two was altered.
- "signature invalid (prev-hash, timestamp, id, or signature altered)" -- the stored
  Ed25519 signature does not verify against the current values of the four signed
  fields; at least one was changed after the block was committed.

**CHAIN BREAK** -- the block's own content and signature are intact, but its chain
link to the preceding block is broken. Three detail strings are possible:

- "chains to altered block N" -- the preceding block is TAMPERED and its content hash
  changed, so this block's stored previous_block_hash no longer matches. This is the
  normal cascade case.
- "does not chain to prior block (deleted or reordered)" -- the preceding block has
  no detected own-block tampering, but the link is still broken; a block may have
  been removed from the database or the sequence may have been reordered.
- "expected genesis block, predecessor missing" -- the first block in the retrieved
  sequence does not carry "genesis" as its previous_block_hash.

### Cascade

Editing one block's content hash makes that block TAMPERED. The next block in the
chain was committed pointing at the original hash; its previous_block_hash no longer
matches. That next block receives CHAIN BREAK -- detail "chains to altered block N".
The TAMPERED block is the root cause; the CHAIN BREAK on the following block is the
downstream effect. This precedence is intentional: the block that actually changed is
named TAMPERED, and downstream breaks point back to it by sequence number.

---

## How a Third Party Verifies an Export

The 7z case export contains `chain_verification.json`. Each block entry carries:

| Field | Content |
|-------|---------|
| `block_id` | UUID of the block |
| `sequence` | 1-based position in chain order |
| `committed_at` | RFC3339 UTC commit timestamp (second precision) |
| `content_hash` | SHA-256 hex digest of the plaintext |
| `previous_block_hash` | Content hash of the preceding block, or "genesis" |
| `signature` | Base64-encoded raw Ed25519 signature (64 bytes, 88 base64 chars) |
| `verdict` | Lowercase: "verified", "tampered", or "chain_break" |
| `detail` | Human-readable reason for non-verified verdicts; empty when verified |

The top-level `examiner_public_key` field is the hex-encoded Ed25519 public key (32
bytes = 64 hex characters).

The archive also contains the decrypted block content as markdown files in
`master_notes/` and `evidence/[ITEM]/`. Each markdown file begins with an HTML
comment header that includes the block_id, content_hash, and previous_block_hash,
followed by a blank line, then the plaintext content.

### Verification steps

**Step 1: Content hash.** For each block, locate the corresponding markdown file
(match by block_id in the file header). Extract the plaintext: the bytes after the
closing `-->` and blank line, without the trailing newline the export format appends.
Compute SHA-256 of those bytes. The result must match the `content_hash` field in
chain_verification.json.

**Step 2: Signature.** Rebuild the canonical signing payload by joining four fields
with a single 0x1F byte between each:

    content_hash + 0x1F + previous_block_hash + 0x1F + committed_at + 0x1F + block_id

All four values come from chain_verification.json for the block in question. The
resulting byte sequence is the message that was signed. Base64-decode the `signature`
field to get the raw 64-byte signature. Hex-decode `examiner_public_key` to get the
32-byte Ed25519 public key. Verify the signature against the message using standard
Ed25519 verification. The result is true or false.

**Step 3: Chain links.** For each block at sequence N > 1, confirm that its
`previous_block_hash` matches the `content_hash` of the block at sequence N-1. The
block at sequence 1 must have `previous_block_hash` equal to "genesis".

A block that passes all three checks is consistent with what the examiner committed
and has not been altered since.

### Example (Python pseudocode)

```python
import hashlib, base64, binascii, json
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey

cv = json.load(open("chain_verification.json"))
pub_bytes = binascii.unhexlify(cv["examiner_public_key"])
pub_key = Ed25519PublicKey.from_public_bytes(pub_bytes)

SEP = b"\x1f"

for block in cv["blocks"]:
    payload = (
        block["content_hash"].encode()   + SEP +
        block["previous_block_hash"].encode() + SEP +
        block["committed_at"].encode()   + SEP +
        block["block_id"].encode()
    )
    sig = base64.b64decode(block["signature"])
    try:
        pub_key.verify(sig, payload)
        sig_ok = True
    except Exception:
        sig_ok = False
    print(f"Block {block['sequence']}: sig={'OK' if sig_ok else 'FAIL'}")
```

---

## What the Signature Does and Does Not Attest

The Ed25519 signature attests that the content hash, chain position (via the
previous-block hash), commit timestamp, and block id were all present with those
exact values when the block was committed by the holder of the private key
corresponding to `examiner_public_key`.

The signature does not establish:

**Author identity beyond the key.** In the current single-user model, the key
belongs to one examiner identity per installation. There is no multi-user key
separation; the signature does not distinguish between analysts who may have shared
an installation.

**The cause of a mismatch.** A failed check means the stored record no longer
matches what was committed. It does not distinguish between deliberate tampering,
storage corruption, or an application defect. The check is a consistency test, not
a forensic determination of intent.

**Third-party key attestation.** The examiner public key is self-generated on first
launch. There is no certificate authority or external attestation binding the key
to the examiner's identity. If the key itself is under challenge, the examiner must
provide it through a separate trusted channel.

---

## Deferred

Public key export as a dedicated feature is pending. The examiner_public_key is
already present in chain_verification.json, so a third party who has the export
can verify independently using the steps above.

Multi-user key separation (signing the author ID as part of the payload) is also
deferred. In the current model, a single keypair covers all blocks committed by
the installation.
