# dfnotes-go

**Version 0.10.2**

A cross-platform desktop application for recording and managing case notes during digital forensic investigations. Built with Go (Wails v2) and React, dfnotes-go provides a structured, tamper-evident note-taking system with a verifiable chain of custody for all entries.

---

## What It Does

dfnotes-go is built around a core principle: every note you write during an investigation should be defensible in legal proceedings. Every note block is hashed (SHA-256), digitally signed (Ed25519), encrypted (AES-256-GCM), and chained to the previous block. If anyone tampers with a block after it is committed, the chain breaks and the app shows it.

Beyond integrity, it handles the practical side of forensic case work:

- Per-case encrypted storage with a separate case password
- Evidence item tracking with configurable numbering and chain of custody logging
- Automated IOC detection (12 types) with confirm/false positive/promote workflow
- Manual IOC and Case Fact creation from selected text in committed block views
- Case Facts tab for informational artifacts that are not IOCs
- Case timeline for key events
- Task list with templates and note linking for investigation workflow tracking
- Documentation reminder timer to prevent examiner tunnel vision
- Encrypted export and automated backup
- Built-in user guide (Help > User Guide)

---

## Features

### Case Management
- Case creation with classification level (Unclassified through Top Secret), ticket number, examiner info
- Classification level editable from the Case Overview tab at any time; changes are audit logged
- Attorney-Client Privilege flag: set at case creation or toggled from Case Overview at any time; changes are audit logged; privileged cases show an amber badge in Case Overview and a Privileged badge on the dashboard case card
- Per-case encryption key derived from a case password via Argon2id
- Case lock/unlock -- walk away from your workstation without exposing case data
- Multiple cases per installation, each independently encrypted

### Note Taking
- Markdown editor with live preview
- Block commit workflow: write, preview, commit -- committed blocks are read-only and tamper-evident
- Amendment workflow: original block stays intact, amendment references the original with a required reason
- Global hash chain per case: every committed block chains to the previous one regardless of which tab it came from

### Evidence Management
- Evidence item registration with type, acquisition hash, status
- Configurable evidence numbering per case: set a prefix (e.g. DF-2025-) and digit count at case creation; default is E001, E002...; prefix accepts alphanumeric characters, hyphens, and underscores; live preview shown during case creation
- Status lifecycle: Collected, Analyzing, Processed, Archived, Withdrawn
- Automatic chain of custody entries on status change, plus manual entries
- Soft delete via Withdrawn status -- no hard deletes
- Current location field: free-text field recording where evidence physically is right now; every change is auto-logged to the chain of custody with the old and new values
- Archive location field: free-text field recording final disposition after a case closes; prompted via modal when an item is set to Archived (with a "Set Later" option); an amber badge appears in the detail view and list when an item is Archived but archive location has not been set; editable at any time from the detail view; changes do not generate custody log entries
- Dynamic evidence tabs (E001, E002, etc.) each with their own note editor
- Evidence linking in markdown using `[[E001]]` syntax (or any custom format like `[[DF-2025-001]]`) with autocomplete

### IOC Detection
- Automatic detection on block commit for 12 IOC types: IPv4, IPv6, domain, URL, email, MD5, SHA1, SHA256, Windows file path, Unix file path, registry key, CVE
- Manual "file" type for standalone filenames (e.g. NTUSER.DAT) -- not auto-detected, assignable via type correction
- Manual IOC creation from selected text: highlight any text in a committed block, right-click, choose "Mark as IOC" -- type is pre-selected via pattern matching, value is pre-filled and editable
- Yellow highlight for detected IOCs, red for confirmed; false positive and promoted IOCs render as plain unstyled text in block view
- Right-click context menu to confirm, dismiss, promote to Case Facts, or restore status
- IOC type editable from the IOC Summary tab via an inline dropdown -- useful when auto-detection misclassifies a value
- Status or type changes in the IOC Summary tab immediately update highlights in all committed block views
- IOC Summary tab with defanged display, filtering by type and evidence item, source navigation
- Promoted IOCs appear in a separate section at the bottom of IOC Summary with a Restore action
- All IOC values stored raw, defanged only on display

### Case Facts
- Dedicated tab (after Case Overview, before Master Notes) for informational investigation artifacts
- Stores items like subject usernames, machine hostnames, IP addresses, OS versions, timezones, and similar reference data that are informational rather than indicators of compromise
- 16 predefined types (username, hostname, IP address, MAC address, OS version, timezone, email address, account SID, full name, phone number, device serial, URL, file path, domain, registry key, custom)
- Each fact has a type, description, value, optional evidence item association, optional source block link, and optional notes
- Quick entry from committed block view: highlight text, right-click, choose "Add as Case Fact" to open the add form pre-populated with the selection
- IOCs can be promoted directly to Case Facts from the IOC Summary tab or from the right-click context menu in block view -- promotion sets the IOC to a "promoted" status (plain text in block view) and creates the fact with provenance back to the source IOC
- Promoting an IOC is reversible: Restore in the promoted section of IOC Summary deletes the case fact and returns the IOC to detected status
- Filterable by type and evidence item, combinable
- Audit logged on create, update, and delete; not immutable
- Included in both the 7z export archive (case_facts.json) and the PDF export

### Timeline
- Manual timeline entries with ISO 8601 UTC timestamp (required) and optional secondary IANA timezone display
- Searchable timezone dropdown with city name and UTC offset lookup
- Sortable by timestamp
- Source navigation links to the originating evidence tab

### Task List
- Per-case task tracking with five statuses: Open, In Progress, Blocked, Complete, Not Applicable
- Completion timestamp recorded when a task is marked Complete
- Tasks assigned to a specific evidence item or to the case overall
- Task templates: named task sets configured in Settings, applied at any point during an investigation with evidence item assignment
- Many-to-many note block linking: link committed note blocks to tasks as documentation of the work done
- Filtering by status and evidence item, combinable
- Tasks included in case export

### Documentation Reminder
- Countdown timer that alerts the examiner when too much time has passed without documenting something
- Runs only while a case is unlocked; stopped on lock
- Full reset on committed note block or timeline entry add/edit; partial reset (half interval) on evidence status change or manual custody log entry
- Reminder modal comes to the foreground when the timer fires; four options: Document Now, Snooze 15 min, Snooze 30 min, Pause reminders
- Pause state shown via a persistent banner with a Resume button; clears on case lock
- Configurable from Settings: enable/disable toggle and interval in minutes (default 30)
- Settings visit does not interrupt the timer or lock the case

### Tagging
- 28 predefined standard tags across analysis, status, priority, and evidence type categories
- Custom tag creation per case
- Tags on both note blocks and evidence items
- Tag filtering on notes and evidence list

### Theming
- 11 themes selectable from View > Theme: Forensic Dark (default), Classic Dark, High Contrast, Light, Solarized Dark, Monokai, Dracula, Nord, Gruvbox, Matrix, Forensic Blue
- Applied immediately, persisted across restarts

### Backup
- Automated encrypted backups: raw copy of the SQLite database (content is already AES-256-GCM encrypted at the application layer)
- Configurable destination, interval (default 6 hours), and retention count
- Backup filenames include ISO 8601 UTC timestamps: `dfnotes-go_backup_YYYYMMDDTHHMMSSZ.db`
- Persistent failure notification banner with snooze and dismiss options
- Manual "Back up now" trigger in Settings
- Last backup timestamp and status survive application restarts

### Export
- Full case export to an AES-256 encrypted 7z archive
- User sets a separate archive password at export time (independent of the case password)
- User selects the save location via a native file dialog
- Archive contents:
  - `README.txt` -- description and verification instructions
  - `case_metadata.json` -- case metadata in plaintext
  - `[CASENUMBER].db` -- the encrypted SQLite database
  - `master_notes/` -- one markdown file per committed block with hash/signature header
  - `evidence/[ITEM]/` -- metadata.json (includes current_location and archive_location) and block markdown files per evidence item
  - `ioc_summary.json` -- all IOCs with raw and defanged values
  - `case_facts.json` -- all case facts with type, description, value, and source references
  - `timeline.json` -- all timeline entries
  - `tasks.json` -- all tasks with status, evidence item, and linked block references
  - `chain_verification.json` -- full hash chain with per-block verification results (content hash, previous-block hash, commit timestamp, block id, base64 signature, verdict, and detail) and top-level `chain_intact` flag; contains all fields needed for independent re-verification
- Once extracted, archive contents are not encrypted -- handle according to your organization's data handling policy
- Export logged in the audit trail

### PDF Export
- Full case PDF export via File > Export PDF
- Classification level header and footer on every page, color-coded by level
- Sections: cover page, table of contents, case facts, master notes, evidence items, IOC summary, timeline, task list, chain verification, image appendix
- Note blocks rendered with full verification hash (SHA-256), commitment timestamp, and block ID; per-block headers show the real verification verdict from the chain run
- Chain verification section: explanatory statement, chain-intact summary line, per-block table (hash, signature, link, verdict), and a findings list for any failures
- IOC values displayed defanged throughout; case fact values displayed raw
- SHA-256 sidecar file written alongside the PDF for integrity verification
- Export logged in the audit trail

### Database Location
- Database location is configurable at first launch and changeable at any time in Settings
- Move: closes connection, copies file, verifies integrity via SHA-256, deletes original, reopens at new path
- Point: validates schema of an existing database file, switches to it without touching the original
- Config file stored at `~/.config/dfnotes-go/config.json` (Linux), separate from the database

### Security
- User identity with Ed25519 keypair generated on first launch
- Optional TOTP MFA with QR code enrollment and one-time recovery codes
- Master key derived from application password via Argon2id
- Per-case encryption keys wrapped by the master key
- All note block content encrypted with AES-256-GCM

### Chain Verification
- Button-triggered integrity check from the Chain Verification tab (far right of the tab bar)
- Decrypts each block, recomputes its SHA-256 content hash, and verifies the Ed25519 signature over the canonical payload (content hash, previous-block hash, commit timestamp, and block id, joined by ASCII Unit Separator 0x1F)
- Confirms each block's chain link to its predecessor
- Verdict banner plus a per-block table in chain order; failed rows highlighted; click any row to navigate directly to that block
- Per-block verdicts: VERIFIED; TAMPERED with reason (decryption failed, content hash mismatch, signature invalid); CHAIN BREAK with detail (chains to altered block N, does not chain to prior block)
- Each run is logged as a VERIFY audit entry
- The same engine drives the live note badges, the Chain Verification tab, the 7z export chain_verification.json, and the PDF chain verification section -- all surfaces use identical logic

### User Guide
- Built-in help accessible from Help > User Guide
- 15 sections covering all features
- Fully theme-aware

---

## Requirements

### Runtime
- Linux (amd64), Windows (amd64), or macOS (arm64)
- For export: the `7z` command-line tool must be installed
  - Linux: `sudo apt install p7zip-full`
  - Windows: [7-Zip](https://www.7-zip.org/)
  - macOS: `brew install p7zip`

> **Note for Linux users:** The Ubuntu Archive Manager (file-roller) and similar GUI archive tools do not support AES-256 encrypted 7z archives. Use the 7z command line to open exports:
> ```
> 7z x -p[yourpassword] export_file.7z -o./output_dir/
> ```

### Build Dependencies (if building from source)
- Go 1.21+
- Node.js 18+
- Wails v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Linux: `webkit2gtk-4.1` development libraries

---

## Building from Source

```bash
# Clone the repository
git clone https://github.com/CdtDelta/dfnotes-go.git
cd dfnotes-go

# Build (Linux)
wails build -tags webkit2_41

# Build (Windows/macOS)
wails build

# Binary location
build/bin/dfnotes-go
```

---

## Known Limitations

- **Image clipboard paste** does not work on Linux due to WebKit2GTK Bug 218519. Use the "Attach Image" button in the editor toolbar to attach images via the native file dialog.
- **IOC detection in code spans:** IOC patterns match content inside inline code spans (backtick-wrapped text). A hash or CVE inside a code span will be detected and highlighted.
- **Unix file paths** are detected by the backend and appear in the IOC Summary tab but are not highlighted in the committed block view (too many false positives in rendered markdown).
- **Manual IOC selection across formatting boundaries:** If a text selection spans a markdown formatting boundary (e.g. a bold tag splitting a filename), the stored value is correct but the highlight pipeline may not match it in the rendered HTML. The IOC will still appear correctly in IOC Summary.
- **Export requires 7z:** The export feature shells out to the system `7z` binary. If it is not installed, the export will fail with a clear error message. See Requirements above.
- **Archive manager compatibility:** AES-256 encrypted 7z archives must be opened with the 7z CLI or 7-Zip. Most GUI archive managers including Ubuntu's file-roller do not support them.
- **Document Now focus:** The documentation reminder modal's Document Now button focuses the first textarea in the DOM (80ms delay) rather than targeting the editor via a named ref. Works correctly in practice since the notes editor is the only textarea present when the modal fires.
- **PDF bold text:** The PDF export renders bold text in regular weight. DejaVu Sans Bold is not yet embedded; bold style falls back to regular weight throughout the PDF.
- **Evidence item cross-referencing in export JSON:** `ioc_summary.json`, `case_facts.json`, and the markdown block file header comments still reference evidence items by raw `evidence_item_id` (UUID) rather than the configured item_number. `metadata.json` and `tasks.json` do carry the human-readable number.

---

## Changelog

### v0.10.2 (2026-07-31)

**Evidence Export Numbering Fix**
- PDF export (Evidence Items section, Timeline source column, Task List
  evidence column) and 7z export (tasks.json evidence labels) now use
  each case's configured evidence item_number instead of a positional
  E### label derived from creation order. Cases using a custom evidence
  prefix (e.g. DF-2025-) previously showed the default E001 format in
  exports regardless of configuration
- 7z export evidence metadata.json now includes an item_number field,
  giving each evidence item's configured number a canonical place in
  the export archive (previously only identifiable by the raw
  evidence_item_id UUID)

### v0.10.1 (2026-07-15)

**Evidence Item Identification**
- Evidence item number (e.g. E001, DF-2025-001) now displayed as a badge
  next to the item name in the evidence list and in the evidence detail
  view
- Evidence item note tabs now show a static header combining the item
  number and name above the note editor, so the tab's content is
  self-identifying without needing to cross-reference the evidence list

### v0.10.0 (2026-06-29)

**Chain Verification Engine**
- New `internal/verify` package with real per-block verification: decrypt each block, recompute the SHA-256 content hash, verify the Ed25519 signature over the canonical payload, and confirm chain linkage to the preceding block
- Block signatures now cover a canonical payload of four fields joined by ASCII Unit Separator (0x1F): content hash, previous-block hash, commit timestamp (RFC3339 UTC, second precision), and block id. Signatures previously covered the content hash only; blocks committed before 0.10.0 will show signature-invalid
- New Chain Verification tab (far right of the tab bar): button-triggered, ephemeral result; verdict banner plus a full per-block table in chain order; failed rows highlighted; click a row to navigate to the source block; run logged as a VERIFY audit entry
- Per-block verdict vocabulary: VERIFIED; TAMPERED with reason (decryption failed, content hash mismatch, signature invalid); CHAIN BREAK with detail (chains to altered block N, does not chain to prior block, expected genesis block)
- 7z export chain_verification.json now carries real per-block results plus the raw materials for independent re-verification: content_hash, previous_block_hash, committed_at, block_id, and base64-encoded signature, alongside verdict and detail
- PDF chain verification section now renders real verdicts from the same engine; per-block headers in Master Notes and Evidence sections show the real verdict, so a chain-break block no longer prints VERIFIED in its own header
- All surfaces (live note badges, Chain Verification tab, 7z export, PDF) run the same `internal/verify` engine and cannot disagree

### v0.9.0 (2026-06-26)

**Evidence Location Fields**
- Two new fields on every evidence item: Current Location (where the evidence is now) and Archive Location (final disposition after case closes)
- Current Location: free-text, editable at any time from the detail view; every change auto-logs a chain of custody entry with old and new values (e.g. "Location updated: Evidence locker B-12 -> Examiner workstation 3")
- Archive Location: free-text, editable at any time; prompted via modal when an item transitions to Archived status; modal includes a "Set Later" option that closes without saving; changes do not generate custody log entries
- Amber warning badge shown in the evidence detail view header and as an indicator on the evidence list when an item is Archived but archive location has not been recorded; badge clears immediately when the field is populated
- Withdrawn status does not trigger the archive location modal
- Migration 014 adds current_location and archive_location columns to evidence_items

**Export Format Updates**
- Both location fields are now included in evidence metadata.json inside the 7z archive export (present even when empty, as empty strings)
- PDF evidence items section renders Current Location and Archive Location rows when non-empty; no blank rows for unset fields

**PDF UTF-8 Font**
- Replaced the Latin-1 default font with embedded DejaVu Sans via go:embed and AddUTF8FontFromBytes
- Covers Latin Extended, Cyrillic, and Greek character ranges; CJK is a separate future decision
- Bold text currently renders in regular weight (DejaVu Sans Bold not yet embedded); all bold SetFont calls fall back to regular style

### v0.8.5 (2026-06-12)

**Manual IOC and Case Fact from Selection**
- Highlight any text in a committed note block (Master Notes or any evidence tab), right-click, and choose "Mark as IOC" or "Add as Case Fact"
- "Mark as IOC" opens a modal with the selected text pre-filled as the value; type is pre-selected via pattern matching (URL before domain, SHA256 before SHA1 before MD5); type is required if no pattern matches
- Manually created IOCs are stored with detection_method = "manual" and status = "confirmed"; the highlight appears in the block view immediately via iocVersion increment -- no tab switch or restart required
- "Add as Case Fact" switches to the Case Facts tab and opens the add form pre-populated with the selected text
- Right-clicking an existing IOC highlight while a selection is active shows both the existing IOC options and the new selection options simultaneously
- Duplicate manual IOC submissions (same block, type, and value) return a clear error message

### v0.8.0 (2026-06-12)

**Case Facts**
- New Case Facts tab (positioned after Case Overview, before Master Notes) for storing informational investigation artifacts that are not IOCs -- subject usernames, machine hostnames, IP addresses, OS versions, timezones, and similar reference data
- Each fact has a type (16 predefined types plus custom), description, value, optional evidence item association, optional source block link, and optional notes
- Filterable by type and evidence item, combinable with AND logic
- Audit logged on create, update, and delete; not immutable
- Included in the 7z export archive as case_facts.json and in the PDF export as a Case Facts section (between Case Overview and Master Notes)

**IOC Promotion**
- IOCs can now be promoted to Case Facts from the IOC Summary tab row actions or from the right-click context menu in committed block view
- Promotion modal pre-fills type (mapped from IOC type) and value (raw, not defanged); description and any overrides are entered by the analyst
- Promoted IOCs render as plain text in committed block view (same as false positive)
- Promoted IOCs appear in a separate "Promoted to Case Facts" section at the bottom of IOC Summary, distinct from the false positives section
- Restore action in the promoted section deletes the associated case fact and returns the IOC to detected status (yellow highlight) immediately via iocVersion increment
- Migrations 012 (case_facts table) and 013 (ioc_entries rebuilt with promoted in CHECK constraint)

### v0.7.1 (2026-06-10)

**Bug Fixes**
- Task list evidence item labels now use stored item_number values directly instead of deriving a positional E### label from sort order. Affected the task row evidence column, task detail panel linked note sources, and all evidence assignment dropdowns (filter toolbar, add task form, detail panel, apply template modal)

### v0.7.0 (2026-06-09)

**IOC Improvements**
- False positive IOCs no longer render with a strikethrough in committed block view -- they display as plain text. FP status is still tracked and manageable from the IOC Summary tab with Show False Positives enabled
- Restoring an IOC from false positive to detected in the Summary tab now immediately updates the highlight in all committed block views without requiring a tab switch or restart
- IOC type is now editable from the IOC Summary tab via an inline dropdown, useful when auto-detection misclassifies a value (e.g. a filename detected as a domain name)
- New "file" IOC type for standalone filenames (e.g. NTUSER.DAT), defanged with [.] before the extension; only reachable via manual type correction, not auto-detected

**Case Management**
- Classification level is now editable from the Case Overview tab; changes are audit logged
- Attorney-Client Privilege flag added: set at case creation or toggled at any time from Case Overview; changes are audit logged; privileged cases display an amber badge in Case Overview and a Privileged badge on the dashboard case card

**Evidence Numbering**
- Evidence items can now be numbered with a custom prefix and digit count, configured per case at creation time with a live preview
- Default format remains E001, E002... -- no change to existing behavior
- Prefix accepts alphanumeric characters, hyphens, and underscores
- `[[item_number]]` linking and autocomplete now use stored item numbers directly, supporting arbitrary formats like `[[DF-2025-001]]`

### v0.6.0 (2026-05-23)

**Documentation Reminder**
- Countdown timer that fires a modal when the examiner has not documented for the configured interval
- Full reset on committed note block or timeline entry add/edit; partial reset (half interval, minimum 1 minute) on evidence status change or manual custody log entry; task status changes do not reset the timer
- Modal brings the application window to the foreground; four buttons: Document Now, Snooze 15 min, Snooze 30 min, Pause reminders; no dismiss button
- Snooze resumes at full interval after the snooze period; pause is session-scoped and clears on case lock
- Pause indicator banner with Resume button
- Configurable from Settings: enable/disable toggle and interval in minutes (default 30); inline advisory shown when interval is below 30 minutes

**Settings Modal**
- Settings is now a modal overlay rather than a full-page route, so the active case stays unlocked and the documentation reminder timer keeps running during a Settings visit
- Backdrop click closes Settings only when no unsaved changes are present; Escape key also closes

### v0.5.0 (2026-05-21)

**PDF Export**
- Full case PDF export via File > Export PDF
- Classification level displayed as a color-coded header and footer bar on every page
- Cover page with case metadata, examiner info, export timestamp, and dfnotes-go version
- Table of contents with page numbers
- Master Notes section: all committed blocks in chain order, each with verification hash, commitment timestamp, block ID, and verification status; TAMPERED blocks shown in red
- Evidence Items section: per-item metadata, custody log, and committed note blocks
- IOC Summary section: confirmed and detected IOCs in a monospace table with defanged values; dismissed IOCs in a separate sub-table in gray
- Timeline section: record-block format with timestamp, optional secondary timezone, event description, and investigator notes
- Task List section: grouped by evidence item then by status; each task shows title, status, description, completion timestamp if applicable, and linked block references
- Chain Verification section: explanatory statement, chain-intact summary line, per-block validation table in monospace with red text for any failed rows
- Appendix A: all image attachments from committed note blocks
- SHA-256 sidecar file written alongside the PDF

### v0.4.5 (2026-05-19)

**Task List**
- New Tasks tab per case (positioned after Timeline, before Evidence Tracking)
- Five task statuses: Open, In Progress, Blocked, Complete, Not Applicable
- Completion timestamp recorded when a task is marked Complete
- Tasks assignable to a specific evidence item or to the case overall
- Task templates: named task sets stored in `~/.config/dfnotes-go/templates.json`, managed from Settings, applied from the Task List tab with evidence item assignment
- Many-to-many note block linking via junction table -- link notes to tasks and tasks to notes from either side
- Source navigation from linked notes and linked tasks with pulse animation
- Filtering by status and evidence item, combinable
- Tasks included in case export as `tasks.json`

**User Guide**
- Built-in help dialog accessible from Help > User Guide
- 12 sections covering all application features
- Fully theme-aware, no hardcoded colors

### v0.4.0 (2026-05-18)

**Config and Settings**
- Application configuration now stored in a dedicated config file at an OS-conventional path
- New Settings panel (File > Settings) for backup configuration, database location, template management, and about
- Unsaved changes indicator in Settings panel
- Database path now choosable at setup wizard

**Theming**
- 11 themes available via View > Theme
- Active theme applied immediately and persisted to config
- All colors defined as CSS custom properties -- fully theme-aware including IOC highlights
- Bordered pill-style tabs with accent color on active tab

**Database Location**
- Database location configurable at setup and changeable at any time from Settings
- Move: safely relocates the database with SHA-256 integrity verification and robust error recovery
- Point: switch to a different existing dfnotes-go database without touching the original

**Backup**
- Automated encrypted backups on configurable interval (default 6 hours)
- Configurable destination, interval, and retention count
- Last backup timestamp and status persisted in config
- Manual "Back up now" trigger in Settings
- Persistent failure notification banner with snooze and dismiss

**Export**
- Full case export to AES-256 encrypted 7z archive
- Separate archive password and native save location dialog
- Export logged in the audit trail
- Requires `p7zip-full` on Linux

### v0.3.0 (2026-05-15)

- IOC auto-detection (12 types) on block commit
- IOC highlighting with right-click confirm/false positive workflow
- IOC Summary tab with defanging, filtering, sorting, source navigation
- Timeline tab with ISO 8601 UTC timestamps and optional secondary timezone display
- Searchable timezone dropdown

### v0.2.0

- Evidence item registration and tracking
- Evidence status lifecycle and chain of custody logging
- Dynamic evidence item tabs (E001, E002, etc.)
- Evidence linking in markdown with [[E001]] syntax
- Standard and custom tagging system
- Tag filtering on notes and evidence
- Image attachment via native file dialog

### v0.1.0

- User identity with Ed25519 keypair generation
- Optional TOTP MFA
- Case creation with per-case password and encryption key derivation
- Markdown editor with live preview
- Block commit workflow with SHA-256 hashing and Ed25519 signing
- Global hash chain per case
- Tamper detection with verified/unverified badges
- Audit log

---

## Verification

See [VERIFICATION.md](VERIFICATION.md) for a full description of the integrity model: what each committed block carries, what the verification checks, the verdict taxonomy, and how a third party can independently re-verify an export using chain_verification.json and the examiner public key without running dfnotes-go.

---

## Related Projects

- **4n6time-go** -- Forensic timeline analysis app (same tech stack). Future integration planned to push timeline events from Plaso analysis into dfnotes-go case notes.
- **ezt-go** -- GUI wrapper for Eric Zimmerman's EZ Tools
