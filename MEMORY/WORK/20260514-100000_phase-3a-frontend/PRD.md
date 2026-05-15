---
task: Implement Phase 3a frontend IOC highlighting summary
slug: 20260514-100000_phase-3a-frontend
effort: advanced
phase: execute
progress: 0/50
mode: interactive
started: 2026-05-14T10:00:00Z
updated: 2026-05-14T10:01:00Z
---

## Context

Phase 3a frontend work on top of the completed Go backend. Four deliverables:
1. iocTypes.ts + iocPatterns.ts (mirror Go regex patterns)
2. Post-render DOM pass highlighting in NoteBlockCard
3. IOCContextMenu.tsx right-click status changer
4. IOCSummaryTab.tsx with defang, filtering, sorting

Key discoveries:
- No rehype-* or HAST packages installed; react-markdown 8 uses them internally but they are not exposed. User explicitly approved HTML-string-pass (DOM walk) alternative.
- App.d.ts and App.js are Wails-generated; need manual IOC stubs added (regenerated on next Wails build).
- models.ts is generated; IOCEntry lives in local iocTypes.ts to avoid overwrite.
- NoteBlockCard renders MarkdownRenderer inside a div -- can add a ref there for DOM highlighting.
- CaseDetailPage tabs array: overview, evidence, evidence-notes-*, notes. IOC Summary goes after evidence-notes-* before notes.
- No utils/ directory yet -- create it.

### Highlighting approach
Post-render DOM walk via `useEffect` (no dep array -- re-runs after every render). `applyIOCHighlights` first removes existing `.ioc-highlight` spans (restoring plain text nodes via `normalize()`), then walks `TreeWalker(SHOW_TEXT)` to find and wrap IOC values in spans. Idempotent so safe to call on every render. Status changes trigger re-fetch -> new iocs state -> effect re-applies.

### Wails bindings stubs (manual)
GetBlockIOCs(blockId: string): Promise<ioc.IOCEntry[]>
GetCaseIOCs(caseId: string, includeAll: boolean): Promise<ioc.IOCEntry[]>
UpdateIOCStatus(iocId: string, status: string): Promise<void>

## Criteria

### Types and patterns
- [ ] ISC-1: `iocTypes.ts` defines IOCType union type covering all 11 values
- [ ] ISC-2: `iocTypes.ts` defines IOCStatus union type (detected, confirmed, false_positive)
- [ ] ISC-3: `iocTypes.ts` defines IOCEntry interface with all 12 fields (ioc_id, case_id, block_id, evidence_item_id, type, value, status, detection_method, notes, created_at, confirmed_at, user_id)
- [ ] ISC-4: `iocPatterns.ts` exports IOCPattern interface with type, regex (with g flag), label fields
- [ ] ISC-5: `iocPatterns.ts` exports IOC_PATTERNS array with all 11 entries
- [ ] ISC-6: URL pattern appears before domain in IOC_PATTERNS (more specific first)
- [ ] ISC-7: SHA256 pattern appears before SHA1, SHA1 before MD5 in IOC_PATTERNS
- [ ] ISC-8: User confirms pattern list before highlighting plugin is built

### Wails bindings
- [ ] ISC-9: App.d.ts has GetBlockIOCs declaration returning Promise with IOCEntry array
- [ ] ISC-10: App.d.ts has GetCaseIOCs and UpdateIOCStatus declarations
- [ ] ISC-11: App.js has GetBlockIOCs, GetCaseIOCs, UpdateIOCStatus stub functions

### CSS
- [ ] ISC-12: `.ioc-highlight` base class with 2px border-radius, 0 2px padding, context-menu cursor
- [ ] ISC-13: `.ioc-highlight--detected` yellow background + bottom border
- [ ] ISC-14: `.ioc-highlight--confirmed` red background + bottom border
- [ ] ISC-15: `.ioc-highlight--false_positive` gray background + line-through + opacity

### highlightIOCs.ts
- [ ] ISC-16: `applyIOCHighlights(container, iocs)` exported from `highlightIOCs.ts`
- [ ] ISC-17: Function removes existing `.ioc-highlight` spans before re-applying (idempotent)
- [ ] ISC-18: Function normalizes text nodes after span removal (merges split adjacent nodes)
- [ ] ISC-19: Function uses `TreeWalker(SHOW_TEXT)` to visit all text nodes
- [ ] ISC-20: Matching text node is split into [before-text, span, after-text] correctly
- [ ] ISC-21: Span has data-ioc-id, data-ioc-type, data-ioc-value, data-ioc-status attributes
- [ ] ISC-22: Span class is `ioc-highlight ioc-highlight--{status}` (using status not type for color)

### IOCContextMenu.tsx
- [ ] ISC-23: Menu renders at mouse cursor coordinates (absolute positioned)
- [ ] ISC-24: Menu displays IOC type (as label) and raw value
- [ ] ISC-25: Menu displays current status
- [ ] ISC-26: Detected state offers "Confirm as IOC" and "Mark as False Positive"
- [ ] ISC-27: Confirmed state offers "Unconfirm" and "Mark as False Positive"
- [ ] ISC-28: False positive state offers "Confirm as IOC" and "Restore to Detected"
- [ ] ISC-29: Calls UpdateIOCStatus on selection
- [ ] ISC-30: Span class updated immediately after status change (optimistic UI)
- [ ] ISC-31: Closes on click outside the menu
- [ ] ISC-32: Closes on Escape key

### NoteBlockCard.tsx
- [ ] ISC-33: Calls GetBlockIOCs(block.block_id) on mount, stores in state
- [ ] ISC-34: Wraps MarkdownRenderer in a div with containerRef
- [ ] ISC-35: Calls applyIOCHighlights in useEffect (no dep array) after each render
- [ ] ISC-36: Right-click on container uses event delegation to detect .ioc-highlight spans
- [ ] ISC-37: Right-click opens IOCContextMenu with the clicked span's data attributes
- [ ] ISC-38: After UpdateIOCStatus succeeds, re-fetches IOCs to update state

### defang.ts
- [ ] ISC-39: ipv4: replaces all dots with [.]
- [ ] ISC-40: ipv6: replaces all colons with [:]
- [ ] ISC-41: domain: replaces only the last dot with [.]
- [ ] ISC-42: url: replaces scheme with hxxp/hxxps/fxp, replaces all dots with [.]
- [ ] ISC-43: email: replaces @ with [@]
- [ ] ISC-44: hash types, cve, file_path, registry_key: returned as-is (no transform)

### IOCSummaryTab.tsx
- [ ] ISC-45: Fetches GetCaseIOCs(caseId, false) on mount (FPs excluded by default)
- [ ] ISC-46: "Show False Positives" toggle re-fetches with includeAll=true
- [ ] ISC-47: All values displayed defanged using defang.ts
- [ ] ISC-48: Type column shows colored badge per IOC type
- [ ] ISC-49: Status column shows status badge (detected/confirmed/false positive)
- [ ] ISC-50: Source column shows evidence item label (E001 format) or "Master Notes"

### CaseDetailPage.tsx
- [ ] ISC-51: IOC Summary tab added with id='iocs', label='IOC Summary'
- [ ] ISC-52: Tab positioned after evidence-notes-* tabs, before notes tab
- [ ] ISC-53: Renders IOCSummaryTab when activeTab === 'iocs'

### Build
- [ ] ISC-54: TypeScript compiles with no errors (vite build)
- [ ] ISC-55: No `any` type used -- all IOC-related code properly typed

## Decisions

- IOCEntry defined locally in iocTypes.ts (not models.ts) to survive Wails build regeneration
- Post-render DOM walk with useEffect (no deps) rather than rehype plugin -- no new packages needed
- applyIOCHighlights is idempotent: always removes existing spans first
- IOC_PATTERNS in iocPatterns.ts are for UI reference; actual matching uses backend IOCEntry values
- Context menu uses event delegation on the block container (not per-span listeners)
- Summary tab: simple table with defanged values, no clickable block navigation for v1

## Verification
