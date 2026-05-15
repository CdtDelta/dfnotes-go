---
task: Implement Phase 3a IOC detection dfnotes-go
slug: 20260514-000000_phase-3a-ioc-detection
effort: advanced
phase: complete
progress: 45/45
mode: interactive
started: 2026-05-14T00:00:00Z
updated: 2026-05-14T00:01:00Z
---

## Context

Phase 3a of dfnotes-go: IOC detection engine, DB migration, repo + service layer, and wiring into CommitNote. Backend only -- no frontend work until backend compiles clean.

Key discoveries from codebase read:
- `internal/ioc/` directory exists but is empty (.gitkeep only)
- `internal/models/ioc.go` exists with old IOCEntry schema (different field names, no status)
- `internal/database/ioc_repo.go` exists with old schema impl (dead code -- not wired in app.go)
- `models/repositories.go` has old IOCRepository interface (dead code)
- `ioc_entries` table exists in migration 001 with old schema (ioc_type, source_block_id, created_by)
- Migration 004 must rename old table, create new one with spec schema
- IOCService will live in `internal/ioc/` (follows dependency inversion: ioc defines interface, database implements it)
- CommitNote insertion point: `note_service.go:199` after `noteBlockRepo.Create()`
- User requested: show commit code before wiring, wait for confirmation

### Risks
- SQLite does not support DROP COLUMN -- migration must rename+recreate table
- Old models/ioc.go and repositories.go IOCRepository interface conflict with new types -- must update both
- Circular import if ioc package imports database -- resolved by defining IOCRepository interface IN ioc package
- Regex over-matching on domain/unix-path patterns generates false positives -- mitigated by conservative patterns
- DetectAndStore must be truly best-effort -- any panic or error must not propagate to CommitNote

## Criteria

### Migration
- [x] ISC-1: File `004_ioc_entries.sql` exists in migrations directory
- [x] ISC-2: Migration renames old `ioc_entries` to `ioc_entries_v0` before recreating
- [x] ISC-3: New `ioc_entries` has `block_id TEXT NOT NULL` column with FK to note_blocks
- [x] ISC-4: New `ioc_entries` has `evidence_item_id TEXT` nullable column with FK
- [x] ISC-5: New `ioc_entries` has `type TEXT NOT NULL` with CHECK on 11 allowed values
- [x] ISC-6: New `ioc_entries` has `status TEXT NOT NULL DEFAULT 'detected'` with CHECK constraint
- [x] ISC-7: New `ioc_entries` has `confirmed_at TEXT` nullable column
- [x] ISC-8: New `ioc_entries` has `user_id TEXT NOT NULL` column
- [x] ISC-9: Migration has UNIQUE constraint on (block_id, type, value) for deduplication
- [x] ISC-10: Old `ioc_entries_v0` is dropped at end of migration

### ioc_types.go
- [x] ISC-11: IOCType constants use lowercase values matching spec (ipv4, sha256, cve, etc.)
- [x] ISC-12: IOCStatus constants defined: detected, confirmed, false_positive
- [x] ISC-13: IOCMatch struct has Type IOCType and Value string fields
- [x] ISC-14: IOCEntry struct has all spec fields with correct json tags
- [x] ISC-15: IOCRepository interface defined in ioc package (dependency inversion)

### detector.go
- [x] ISC-16: DetectIOCs returns IPv4 matches with word boundary protection
- [x] ISC-17: DetectIOCs returns domain matches requiring at least one dot and valid TLD
- [x] ISC-18: DetectIOCs returns URL matches for http/https/ftp schemes
- [x] ISC-19: DetectIOCs returns SHA256 (64 hex) matches before SHA1 (40 hex) before MD5 (32 hex)
- [x] ISC-20: DetectIOCs returns CVE matches matching CVE-YYYY-NNNNN+ format
- [x] ISC-21: DetectIOCs returns registry key matches anchored on HKLM/HKCU/HKCR/HKEY_ prefixes
- [x] ISC-22: DetectIOCs returns email matches (local@domain.tld)
- [x] ISC-23: DetectIOCs returns Windows file path matches (C:\ prefix)
- [x] ISC-24: DetectIOCs returns Unix file path matches requiring min depth 2
- [x] ISC-25: DetectIOCs runs URL before domain (more specific first) to avoid double-matching

### ioc_repo.go (database package)
- [x] ISC-26: IOCRepo.Create inserts into new schema columns correctly
- [x] ISC-27: IOCRepo.GetByBlock returns all IOCEntry records for given blockID
- [x] ISC-28: IOCRepo.ListByCase filters false_positives when includeAll=false
- [x] ISC-29: IOCRepo.UpdateStatus sets confirmed_at when transitioning TO confirmed
- [x] ISC-30: IOCRepo.UpdateStatus clears confirmed_at for non-confirmed transitions
- [x] ISC-31: IOCRepo.ExistsByBlockValueType checks for duplicate before insert

### models updates
- [x] ISC-32: models/ioc.go cleared of old IOCEntry (replaced by ioc.IOCEntry)
- [x] ISC-33: models/repositories.go IOCRepository interface removed (replaced by ioc.IOCRepository)

### ioc_service.go
- [x] ISC-34: IOCService.DetectAndStore stores each IOCMatch as IOCEntry with status=detected
- [x] ISC-35: IOCService.DetectAndStore skips existing block+type+value combinations
- [x] ISC-36: IOCService.GetCaseIOCs returns detected+confirmed when includeAll=false
- [x] ISC-37: IOCService.GetCaseIOCs returns all including false_positive when includeAll=true
- [x] ISC-38: IOCService.UpdateIOCStatus validates status string before update
- [x] ISC-39: IOCService.GetBlockIOCs returns all IOCs for a specific block

### app.go wiring
- [x] ISC-40: App struct has iocService field of type *ioc.IOCService
- [x] ISC-41: App.startup creates iocRepo and iocService before returning
- [x] ISC-42: App.CommitNote calls iocService.DetectAndStore after noteBlockRepo.Create succeeds
- [x] ISC-43: App.CommitNote still returns success even if DetectAndStore returns error
- [x] ISC-44: App.GetCaseIOCs, App.UpdateIOCStatus, App.GetBlockIOCs exposed as Wails bindings

### Build
- [x] ISC-45: `go build ./...` succeeds with zero errors after all changes

## Decisions

- IOCRepository interface lives in `internal/ioc/` package (not models/) to avoid circular imports
- Old models/ioc.go IOCEntry and repositories.go IOCRepository are dead code -- updating both to remove IOC references
- Migration 004 uses rename+recreate pattern since SQLite cannot DROP COLUMN
- UNIQUE(block_id, type, value) constraint prevents duplicate IOC records per block
- IOCService is in `internal/ioc/service.go` (not internal/services/) since types are ioc-package-local
- Best-effort wiring: DetectAndStore errors are logged to stderr, commit still succeeds
- CommitNote insertion point: after noteBlockRepo.Create() at note_service.go:199, before audit log

## Verification

- ISC-1..10: Migration 004 verified: file exists, RENAME present, all new columns present, UNIQUE and DROP confirmed
- ISC-11..15: ioc_types.go: lowercase constants, IOCStatus, IOCMatch, IOCEntry, IOCRepository with GetExistingByBlock
- ISC-16..25: detector.go: all 11 regex patterns defined, URL before domain in evaluation order, SHA256>SHA1>MD5
- ISC-26..31: ioc_repo.go: Create/GetByBlock/ListByCase(includeAll)/UpdateStatus(confirmed_at)/GetExistingByBlock all verified
- ISC-32..33: models/ioc.go = 0 IOC references, models/repositories.go = 0 IOC references
- ISC-34..39: service.go: DetectAndStore uses GetExistingByBlock (batch), DetectionMethodAuto typed constant, all 4 methods present
- ISC-40..44: app.go: iocService field, startup wires iocRepo+iocService, CommitNote calls DetectAndStore best-effort with log, 3 Wails bindings present
- ISC-45: `go build ./...` + `go test ./internal/...` both pass clean
- /simplify invoked via Skill tool -- confirmed, 3 findings fixed (contains/indexOf, N+1 pattern, typed constants)
