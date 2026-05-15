---
task: fix win path ioc tz combobox timeline nav
slug: 20260515-000000_fix-ioc-path-tz-timeline-nav
effort: extended
phase: complete
progress: 29/29
mode: interactive
started: 2026-05-15T00:00:00Z
updated: 2026-05-15T00:01:00Z
---

## Context

Three bug fixes for dfnotes-go:

1. **Windows file path IOC detection**: Go backend reWinPath regex ends with `[^\\/:*?"<>|\r\n]*` which does not exclude spaces -- greedily captures trailing sentence text. Frontend iocPatterns.ts has same bug (affects context-menu detection). Both need `\b` added at end. Unit test required before fix to demonstrate the bug.

2. **Timezone searchable combobox**: Current TimezoneSelect only searches IANA zone names. Needs city-to-timezone lookup table (bundled, no API) and UTC offset search via runtime Intl.DateTimeFormat computation. Dropdown should show IANA name + match note.

3. **Timeline source navigation**: TimelineTab Source column is non-clickable. Need `onNavigate` prop (same pattern as IOCSummaryTab) wired from CaseDetailPage.handleNavigate. "Case Level" entries must remain non-clickable.

### Risks

- `\b` at end of WinPath regex breaks directory-only paths ending in `\` (acceptable tradeoff for file detection use case)
- UTC offset computation via Intl is per-zone O(n) -- acceptable for ~600 zones, cache in ref after first UTC query
- City lookup: multiple cities can map to same zone (deduplicate by zone in results)

## Criteria

- [x] ISC-1: file_path regex in iocPatterns.ts ends with `\b`
- [x] ISC-2: Frontend regex matches `C:\Windows\System32\evil.dll` and stops there
- [x] ISC-3: Frontend regex does not match trailing text after path
- [x] ISC-4: reWinPath in detector.go uses `[^\s\\...]` final segment (excludes whitespace)
- [x] ISC-5: TestDetectIOCs_WinPath unit test exists in detector_test.go
- [x] ISC-6: Test uses sentence input placing path mid-sentence
- [x] ISC-7: Test asserts captured value is exactly `C:\Windows\System32\evil.dll`
- [x] ISC-8: `go test ./internal/ioc/...` passes
- [x] ISC-9: `go build ./...` passes
- [x] ISC-10: CITY_TO_ZONE lookup table exists in TimezoneSelect component
- [x] ISC-11: Table covers US major cities (SF, NY, Chicago, etc.)
- [x] ISC-12: Table covers European cities (London, Paris, Berlin, etc.)
- [x] ISC-13: Table covers Asia-Pacific cities (Tokyo, Sydney, Singapore, etc.)
- [x] ISC-14: Searching "San Francisco" returns America/Los_Angeles
- [x] ISC-15: Searching "London" returns Europe/London
- [x] ISC-16: Searching "Tokyo" returns Asia/Tokyo
- [x] ISC-17: Searching "UTC-4" returns zones currently at UTC-4 offset
- [x] ISC-18: Searching "UTC+5:30" returns zones currently at UTC+5:30 offset
- [x] ISC-19: UTC offset computation uses Intl.DateTimeFormat at runtime (no hardcoded table)
- [x] ISC-20: Dropdown shows IANA zone name as primary label
- [x] ISC-21: City match noted in parentheses next to IANA name
- [x] ISC-22: UTC offset match noted in parentheses next to IANA name
- [x] ISC-23: IANA zone name search still works alongside city/offset search
- [x] ISC-24: TimelineTab has onNavigate prop (optional, typed correctly)
- [x] ISC-25: Evidence item Source cell is a clickable button
- [x] ISC-26: Clicking evidence source calls onNavigate with correct evidence-notes-{id} tab
- [x] ISC-27: "Case Level" Source cell is plain non-clickable text
- [x] ISC-28: CaseDetailPage passes handleNavigate to TimelineTab
- [x] ISC-29: `npm run build` passes clean

## Decisions

## Verification
