# dfnotes-go

**Version 0.4.7**

A cross-platform desktop application for recording and managing case notes during digital forensic investigations. Built with Go (Wails v2) and React, dfnotes-go provides a structured, tamper-evident note-taking system with a verifiable chain of custody for all entries.

---

## What It Does

dfnotes-go is built around a core principle: every note you write during an investigation should be defensible in legal proceedings. Every note block is hashed (SHA-256), digitally signed (Ed25519), encrypted (AES-256-GCM), and chained to the previous block. If anyone tampers with a block after it is committed, the chain breaks and the app shows it.

Beyond integrity, it handles the practical side of forensic case work:

- Per-case encrypted storage with a separate case password
- Evidence item tracking with chain of custody logging
- Automated IOC detection (12 types) with confirm/false positive workflow
- Case timeline for key events
- Task list with templates and note linking for investigation workflow tracking
- Encrypted export and automated backup
- Built-in user guide (Help > User Guide)

---

## Features

### Case Management
- Case creation with classification level (Unclassified through Top Secret), ticket number, examiner info
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
- Status lifecycle: Collected, Analyzing, Processed, Archived, Withdrawn
- Automatic chain of custody entries on status change, plus manual entries
- Soft delete via Withdrawn status -- no hard deletes
- Dynamic evidence tabs (E001, E002, etc.) each with their own note editor
- Evidence linking in markdown using `[[E001]]` syntax with autocomplete

### IOC Detection
- Automatic detection on block commit for 12 IOC types: IPv4, IPv6, domain, URL, email, MD5, SHA1, SHA256, Windows file path, Unix file path, registry key, CVE
- Yellow highlight for detected IOCs, red for confirmed, gray strikethrough for false positives
- Right-click context menu to confirm, dismiss, or restore status
- IOC Summary tab with defanged display, filtering by type and evidence item, source navigation
- All IOC values stored raw, defanged only on display

### Timeline
- Manual timeline entries with ISO 8601 UTC timestamp (required) and optional secondary IANA timezone display
- Searchable timezone dropdown with city name and UTC offset lookup
- Sortable by timestamp
- Source navigation links to the originating evidence tab

### Task List
- Per-case task tracking with five statuses: Open, In Progress, Blocked, Complete, Not Applicable
- Completion timestamp recorded when a task is marked Complete
- Tasks assigned to a specific evidence item or to the case overall
- Task templates: named task sets configured in Settings, applied at any point during an investigation with evidence item assignment. Useful for standard workflows like hard drive imaging or malware triage
- Many-to-many note block linking: link committed note blocks to tasks as documentation of the work done. A task can have multiple linked notes; a note can link to multiple tasks. Useful at report time for tracing what was done and where the details are recorded
- Filtering by status and evidence item, combinable
- Tasks included in case export

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
  - `evidence/[ITEM]/` -- metadata.json and block markdown files per evidence item
  - `ioc_summary.json` -- all IOCs with raw and defanged values
  - `timeline.json` -- all timeline entries
  - `tasks.json` -- all tasks with status, evidence item, and linked block references
  - `chain_verification.json` -- full hash chain with per-block validation results and `chain_intact` flag
- Once extracted, archive contents are not encrypted -- handle according to your organization's data handling policy
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

### User Guide
- Built-in help accessible from Help > User Guide
- 11 sections covering all features: Getting Started, Note Taking, Evidence Management, IOC Detection, Timeline, Task List, Tagging, Backup, Export, Settings, Tips and Shortcuts
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
- **Export requires 7z:** The export feature shells out to the system `7z` binary. If it is not installed, the export will fail with a clear error message. See Requirements above.
- **Archive manager compatibility:** AES-256 encrypted 7z archives must be opened with the 7z CLI or 7-Zip. Most GUI archive managers including Ubuntu's file-roller do not support them.

---

## Changelog

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
- 11 sections covering all application features
- Fully theme-aware, no hardcoded colors
- Escape key and backdrop click to close

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
- Archive contains case metadata, encrypted database, markdown block files, IOC summary, timeline, chain verification report
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

## Related Projects

- **4n6time-go** -- Forensic timeline analysis app (same tech stack). Future integration planned to push timeline events from Plaso analysis into dfnotes-go case notes.
- **ezt-go** -- GUI wrapper for Eric Zimmerman's EZ Tools
