import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import {
    GetTimelineEntries,
    CreateTimelineEntry,
    UpdateTimelineEntry,
    DeleteTimelineEntry,
} from '../../wailsjs/go/main/App';
import { models, services } from '../../wailsjs/go/models';
import ErrorMessage from './ErrorMessage';

interface TimelineTabProps {
    caseId: string;
    evidenceItems: services.EvidenceResponse[];
    onNavigate?: (tab: string, blockId?: string) => void;
}

interface EntryForm {
    timestamp: string;
    eventDescription: string;
    investigatorNotes: string;
    evidenceItemId: string;
    displayTimezone: string;
}

function nowUTC(): string {
    return new Date().toISOString().replace(/\.\d{3}Z$/, 'Z');
}

function emptyForm(): EntryForm {
    return {
        timestamp: nowUTC(),
        eventDescription: '',
        investigatorNotes: '',
        evidenceItemId: '',
        displayTimezone: '',
    };
}

function formatInTimezone(utcTimestamp: string, tz: string): string {
    try {
        return new Intl.DateTimeFormat('en-CA', {
            timeZone: tz,
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            hour12: false,
        }).format(new Date(utcTimestamp));
    } catch {
        return '';
    }
}

function evidenceLabel(evidenceItemId: string, evidenceItems: services.EvidenceResponse[]): string {
    const sorted = [...evidenceItems].sort((a, b) => a.created_at.localeCompare(b.created_at));
    const idx = sorted.findIndex((e) => e.evidence_item_id === evidenceItemId);
    if (idx === -1) return evidenceItemId;
    return `E${String(idx + 1).padStart(3, '0')} - ${sorted[idx].name}`;
}

const INPUT_CLASS = 'w-full px-3 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-100 focus:outline-none focus:border-blue-500 placeholder-gray-600';
const LABEL_CLASS = 'block text-xs text-gray-400 mb-1';

// City name (lowercase) -> IANA timezone zone
const CITY_TO_ZONE: Record<string, string> = {
    // United States - Pacific
    'los angeles': 'America/Los_Angeles',
    'san francisco': 'America/Los_Angeles',
    'seattle': 'America/Los_Angeles',
    'portland': 'America/Los_Angeles',
    'las vegas': 'America/Los_Angeles',
    // Mountain
    'denver': 'America/Denver',
    'salt lake city': 'America/Denver',
    'albuquerque': 'America/Denver',
    'phoenix': 'America/Phoenix',
    // Central
    'chicago': 'America/Chicago',
    'dallas': 'America/Chicago',
    'houston': 'America/Chicago',
    'minneapolis': 'America/Chicago',
    'new orleans': 'America/Chicago',
    'kansas city': 'America/Chicago',
    // Eastern
    'new york': 'America/New_York',
    'miami': 'America/New_York',
    'boston': 'America/New_York',
    'atlanta': 'America/New_York',
    'washington': 'America/New_York',
    'philadelphia': 'America/New_York',
    'detroit': 'America/Detroit',
    // Other US
    'honolulu': 'Pacific/Honolulu',
    'anchorage': 'America/Anchorage',
    // Canada
    'toronto': 'America/Toronto',
    'montreal': 'America/Toronto',
    'vancouver': 'America/Vancouver',
    'calgary': 'America/Edmonton',
    // Europe
    'london': 'Europe/London',
    'dublin': 'Europe/Dublin',
    'lisbon': 'Europe/Lisbon',
    'paris': 'Europe/Paris',
    'amsterdam': 'Europe/Amsterdam',
    'brussels': 'Europe/Brussels',
    'berlin': 'Europe/Berlin',
    'frankfurt': 'Europe/Berlin',
    'munich': 'Europe/Berlin',
    'zurich': 'Europe/Zurich',
    'geneva': 'Europe/Zurich',
    'vienna': 'Europe/Vienna',
    'rome': 'Europe/Rome',
    'milan': 'Europe/Rome',
    'madrid': 'Europe/Madrid',
    'barcelona': 'Europe/Madrid',
    'stockholm': 'Europe/Stockholm',
    'oslo': 'Europe/Oslo',
    'copenhagen': 'Europe/Copenhagen',
    'helsinki': 'Europe/Helsinki',
    'warsaw': 'Europe/Warsaw',
    'prague': 'Europe/Prague',
    'budapest': 'Europe/Budapest',
    'athens': 'Europe/Athens',
    'istanbul': 'Europe/Istanbul',
    'moscow': 'Europe/Moscow',
    'kyiv': 'Europe/Kyiv',
    'bucharest': 'Europe/Bucharest',
    'sofia': 'Europe/Sofia',
    // Middle East / Africa
    'dubai': 'Asia/Dubai',
    'abu dhabi': 'Asia/Dubai',
    'riyadh': 'Asia/Riyadh',
    'tel aviv': 'Asia/Jerusalem',
    'jerusalem': 'Asia/Jerusalem',
    'doha': 'Asia/Qatar',
    'cairo': 'Africa/Cairo',
    'johannesburg': 'Africa/Johannesburg',
    'nairobi': 'Africa/Nairobi',
    'lagos': 'Africa/Lagos',
    // Asia
    'kolkata': 'Asia/Kolkata',
    'mumbai': 'Asia/Kolkata',
    'delhi': 'Asia/Kolkata',
    'new delhi': 'Asia/Kolkata',
    'bangalore': 'Asia/Kolkata',
    'karachi': 'Asia/Karachi',
    'lahore': 'Asia/Karachi',
    'islamabad': 'Asia/Karachi',
    'dhaka': 'Asia/Dhaka',
    'colombo': 'Asia/Colombo',
    'kathmandu': 'Asia/Kathmandu',
    'kabul': 'Asia/Kabul',
    'tehran': 'Asia/Tehran',
    'tashkent': 'Asia/Tashkent',
    'almaty': 'Asia/Almaty',
    'bangkok': 'Asia/Bangkok',
    'hanoi': 'Asia/Bangkok',
    'ho chi minh': 'Asia/Ho_Chi_Minh',
    'jakarta': 'Asia/Jakarta',
    'kuala lumpur': 'Asia/Kuala_Lumpur',
    'singapore': 'Asia/Singapore',
    'manila': 'Asia/Manila',
    'taipei': 'Asia/Taipei',
    'hong kong': 'Asia/Hong_Kong',
    'beijing': 'Asia/Shanghai',
    'shanghai': 'Asia/Shanghai',
    'guangzhou': 'Asia/Shanghai',
    'shenzhen': 'Asia/Shanghai',
    'seoul': 'Asia/Seoul',
    'tokyo': 'Asia/Tokyo',
    'osaka': 'Asia/Tokyo',
    'ulaanbaatar': 'Asia/Ulaanbaatar',
    // Pacific / Australia
    'sydney': 'Australia/Sydney',
    'melbourne': 'Australia/Melbourne',
    'brisbane': 'Australia/Brisbane',
    'perth': 'Australia/Perth',
    'adelaide': 'Australia/Adelaide',
    'auckland': 'Pacific/Auckland',
    'wellington': 'Pacific/Auckland',
    'fiji': 'Pacific/Fiji',
    'guam': 'Pacific/Guam',
    // South / Central America
    'sao paulo': 'America/Sao_Paulo',
    'buenos aires': 'America/Argentina/Buenos_Aires',
    'santiago': 'America/Santiago',
    'lima': 'America/Lima',
    'bogota': 'America/Bogota',
    'caracas': 'America/Caracas',
    'mexico city': 'America/Mexico_City',
};

function formatOffset(minutes: number): string {
    const sign = minutes >= 0 ? '+' : '-';
    const abs = Math.abs(minutes);
    const h = Math.floor(abs / 60);
    const m = abs % 60;
    return `UTC${sign}${h}${m > 0 ? `:${String(m).padStart(2, '0')}` : ''}`;
}

function getZoneOffsetMinutes(zone: string, now: Date): number | null {
    try {
        const opts: Intl.DateTimeFormatOptions = {
            year: 'numeric', month: '2-digit', day: '2-digit',
            hour: '2-digit', minute: '2-digit', second: '2-digit',
            hour12: false,
        };
        const toParts = (tz: string) => {
            const parts = new Intl.DateTimeFormat('en-CA', { ...opts, timeZone: tz }).formatToParts(now);
            const g = (t: string) => +(parts.find((p) => p.type === t)?.value ?? 0);
            return Date.UTC(g('year'), g('month') - 1, g('day'), g('hour'), g('minute'), g('second'));
        };
        return (toParts(zone) - toParts('UTC')) / 60000;
    } catch {
        return null;
    }
}

type ZoneResult = { ianaZone: string; matchNote: string };

function TimezoneSelect({
    value,
    onChange,
    className,
}: {
    value: string;
    onChange: (v: string) => void;
    className?: string;
}) {
    const [query, setQuery] = useState(value);
    const [open, setOpen] = useState(false);
    const ref = useRef<HTMLDivElement>(null);
    const offsetCache = useRef<Map<string, number> | null>(null);

    const allZones = useMemo<string[]>(() => {
        try { return (Intl as unknown as { supportedValuesOf: (k: string) => string[] }).supportedValuesOf('timeZone'); }
        catch { return []; }
    }, []);

    useEffect(() => { setQuery(value); }, [value]);

    useEffect(() => {
        if (!open) return;
        const handler = (e: MouseEvent) => {
            if (ref.current && !ref.current.contains(e.target as Node)) {
                setOpen(false);
                if (!allZones.includes(query)) setQuery(value);
            }
        };
        document.addEventListener('mousedown', handler);
        return () => document.removeEventListener('mousedown', handler);
    }, [open, query, value, allZones]);

    const results = useMemo((): ZoneResult[] => {
        if (!query.trim()) {
            return allZones.slice(0, 100).map((z) => ({ ianaZone: z, matchNote: '' }));
        }
        const q = query.trim();

        // UTC offset search: UTC+5:30, UTC-4, utc+0, etc.
        const utcMatch = q.match(/^utc([+-])(\d{1,2})(?::(\d{2}))?$/i);
        if (utcMatch) {
            const sign = utcMatch[1] === '+' ? 1 : -1;
            const targetOffset = sign * (parseInt(utcMatch[2]) * 60 + parseInt(utcMatch[3] ?? '0'));
            if (!offsetCache.current) {
                const now = new Date();
                offsetCache.current = new Map();
                for (const z of allZones) {
                    const off = getZoneOffsetMinutes(z, now);
                    if (off !== null) offsetCache.current.set(z, off);
                }
            }
            return Array.from(offsetCache.current.entries())
                .filter(([, off]) => off === targetOffset)
                .map(([z]) => ({ ianaZone: z, matchNote: formatOffset(targetOffset) }))
                .slice(0, 100);
        }

        const ql = q.toLowerCase();
        const seen = new Set<string>();
        const out: ZoneResult[] = [];

        // City name search
        for (const [city, zone] of Object.entries(CITY_TO_ZONE)) {
            if (city.includes(ql) && !seen.has(zone)) {
                const label = city.replace(/\b\w/g, (c) => c.toUpperCase());
                out.push({ ianaZone: zone, matchNote: label });
                seen.add(zone);
            }
        }

        // IANA zone name search
        for (const z of allZones) {
            if (z.toLowerCase().includes(ql) && !seen.has(z)) {
                out.push({ ianaZone: z, matchNote: '' });
                seen.add(z);
            }
        }

        return out.slice(0, 100);
    }, [query, allZones]);

    const select = (tz: string) => { onChange(tz); setQuery(tz); setOpen(false); };
    const clear = () => { onChange(''); setQuery(''); setOpen(false); };

    return (
        <div ref={ref} className="relative">
            <input
                type="text"
                value={query}
                placeholder="Search timezone, city, or UTC offset..."
                onChange={(e) => { setQuery(e.target.value); setOpen(true); }}
                onFocus={() => setOpen(true)}
                className={className}
            />
            {open && (
                <div className="absolute z-50 top-full mt-1 w-full bg-gray-800 border border-gray-600 rounded shadow-xl max-h-48 overflow-y-auto">
                    <button
                        onMouseDown={(e) => { e.preventDefault(); clear(); }}
                        className="w-full text-left px-3 py-1.5 text-sm text-gray-500 hover:bg-gray-700 border-b border-gray-700"
                    >
                        -- No timezone --
                    </button>
                    {results.map(({ ianaZone, matchNote }) => (
                        <button
                            key={`${ianaZone}-${matchNote}`}
                            onMouseDown={(e) => { e.preventDefault(); select(ianaZone); }}
                            className={`w-full text-left px-3 py-1.5 text-sm hover:bg-gray-700 transition-colors ${ianaZone === value ? 'text-blue-400 bg-gray-700/50' : 'text-gray-200'}`}
                        >
                            {ianaZone}
                            {matchNote && <span className="ml-2 text-xs text-gray-500">({matchNote})</span>}
                        </button>
                    ))}
                    {results.length === 0 && (
                        <p className="px-3 py-2 text-sm text-gray-500">No matching timezone.</p>
                    )}
                </div>
            )}
        </div>
    );
}

export default function TimelineTab({ caseId, evidenceItems, onNavigate }: TimelineTabProps) {
    const [entries, setEntries] = useState<models.TimelineEntry[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');

    const [form, setForm] = useState<EntryForm>(emptyForm);
    const [formError, setFormError] = useState('');
    const [submitting, setSubmitting] = useState(false);

    const [editingId, setEditingId] = useState<string | null>(null);
    const [editForm, setEditForm] = useState<EntryForm>(emptyForm);
    const [editError, setEditError] = useState('');
    const [saving, setSaving] = useState(false);

    const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

    const sortedEvidenceItems = [...evidenceItems].sort((a, b) =>
        a.created_at.localeCompare(b.created_at)
    );

    const fetchEntries = useCallback(() => {
        setLoading(true);
        GetTimelineEntries(caseId)
            .then((result) => setEntries(result || []))
            .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
            .finally(() => setLoading(false));
    }, [caseId]);

    useEffect(() => {
        fetchEntries();
    }, [fetchEntries]);

    const displayed = [...entries].sort((a, b) => {
        const cmp = a.timestamp.localeCompare(b.timestamp);
        return sortDir === 'asc' ? cmp : -cmp;
    });

    const handleAdd = async () => {
        setFormError('');
        if (!form.eventDescription.trim()) {
            setFormError('Event description is required.');
            return;
        }
        if (isNaN(Date.parse(form.timestamp))) {
            setFormError('Timestamp must be a valid ISO 8601 date-time string.');
            return;
        }
        setSubmitting(true);
        try {
            await CreateTimelineEntry({
                case_id: caseId,
                timestamp: form.timestamp,
                event_description: form.eventDescription.trim(),
                investigator_notes: form.investigatorNotes.trim(),
                evidence_item_id: form.evidenceItemId || undefined,
                display_timezone: form.displayTimezone.trim() || undefined,
            } as models.CreateTimelineEntryRequest);
            setForm(emptyForm());
            fetchEntries();
        } catch (err: unknown) {
            setFormError(err instanceof Error ? err.message : String(err));
        } finally {
            setSubmitting(false);
        }
    };

    const startEdit = (entry: models.TimelineEntry) => {
        setEditingId(entry.entry_id);
        setEditForm({
            timestamp: entry.timestamp,
            eventDescription: entry.event_description,
            investigatorNotes: entry.investigator_notes,
            evidenceItemId: entry.evidence_item_id || '',
            displayTimezone: entry.display_timezone || '',
        });
        setEditError('');
    };

    const handleSaveEdit = async () => {
        if (!editingId) return;
        setEditError('');
        if (!editForm.eventDescription.trim()) {
            setEditError('Event description is required.');
            return;
        }
        if (isNaN(Date.parse(editForm.timestamp))) {
            setEditError('Timestamp must be a valid ISO 8601 date-time string.');
            return;
        }
        setSaving(true);
        try {
            await UpdateTimelineEntry({
                entry_id: editingId,
                timestamp: editForm.timestamp,
                event_description: editForm.eventDescription.trim(),
                investigator_notes: editForm.investigatorNotes.trim(),
                display_timezone: editForm.displayTimezone.trim() || undefined,
            } as models.UpdateTimelineEntryRequest);
            setEditingId(null);
            fetchEntries();
        } catch (err: unknown) {
            setEditError(err instanceof Error ? err.message : String(err));
        } finally {
            setSaving(false);
        }
    };

    const handleDelete = async (entryId: string) => {
        try {
            await DeleteTimelineEntry(entryId);
            setConfirmDelete(null);
            setEntries((prev) => prev.filter((e) => e.entry_id !== entryId));
        } catch (err: unknown) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    return (
        <div className="space-y-6">
            {/* Add Entry Form */}
            <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                <h3 className="text-sm font-medium text-gray-300 mb-4">Add Timeline Entry</h3>
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                    <div>
                        <label className={LABEL_CLASS}>Timestamp (ISO 8601 UTC)</label>
                        <input
                            type="text"
                            value={form.timestamp}
                            onChange={(e) => setForm((f) => ({ ...f, timestamp: e.target.value }))}
                            placeholder="2026-05-14T22:00:00Z"
                            className={INPUT_CLASS}
                        />
                    </div>
                    <div>
                        <label className={LABEL_CLASS}>Display Timezone (optional)</label>
                        <TimezoneSelect
                            value={form.displayTimezone}
                            onChange={(v) => setForm((f) => ({ ...f, displayTimezone: v }))}
                            className={INPUT_CLASS}
                        />
                    </div>
                    <div className="sm:col-span-2">
                        <label className={LABEL_CLASS}>Event Description</label>
                        <input
                            type="text"
                            value={form.eventDescription}
                            onChange={(e) => setForm((f) => ({ ...f, eventDescription: e.target.value }))}
                            placeholder="What happened?"
                            className={INPUT_CLASS}
                            onKeyDown={(e) => { if (e.key === 'Enter') handleAdd(); }}
                        />
                    </div>
                    <div className="sm:col-span-2">
                        <label className={LABEL_CLASS}>Investigator Notes (optional)</label>
                        <textarea
                            rows={3}
                            value={form.investigatorNotes}
                            onChange={(e) => setForm((f) => ({ ...f, investigatorNotes: e.target.value }))}
                            placeholder="Additional context..."
                            className={INPUT_CLASS + ' resize-none'}
                        />
                    </div>
                    <div>
                        <label className={LABEL_CLASS}>Evidence Item (optional)</label>
                        <select
                            value={form.evidenceItemId}
                            onChange={(e) => setForm((f) => ({ ...f, evidenceItemId: e.target.value }))}
                            className={INPUT_CLASS}
                        >
                            <option value="">-- Case Level --</option>
                            {sortedEvidenceItems.map((item, idx) => (
                                <option key={item.evidence_item_id} value={item.evidence_item_id}>
                                    E{String(idx + 1).padStart(3, '0')} - {item.name}
                                </option>
                            ))}
                        </select>
                    </div>
                    <div className="flex items-end">
                        <button
                            onClick={handleAdd}
                            disabled={submitting}
                            className="bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white py-1.5 px-4 rounded text-sm transition-colors"
                        >
                            {submitting ? 'Adding...' : 'Add Entry'}
                        </button>
                    </div>
                </div>
                {formError && (
                    <p className="mt-2 text-xs text-red-400">{formError}</p>
                )}
            </div>

            <ErrorMessage message={error} onDismiss={() => setError('')} />

            {/* Timeline Table */}
            <div>
                <div className="flex items-center justify-between mb-3">
                    <h3 className="text-sm font-medium text-gray-400">
                        Timeline ({entries.length} {entries.length === 1 ? 'entry' : 'entries'})
                    </h3>
                </div>

                {loading ? (
                    <div className="flex justify-center py-8">
                        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-400"></div>
                    </div>
                ) : entries.length === 0 ? (
                    <p className="text-gray-500 text-sm text-center py-8">No timeline entries yet.</p>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full text-sm">
                            <thead>
                                <tr className="text-left border-b border-gray-700">
                                    <th className="pb-2 pr-4 w-48">
                                        <button
                                            onClick={() => setSortDir((d) => d === 'asc' ? 'desc' : 'asc')}
                                            className="flex items-center gap-1 font-medium text-gray-400 hover:text-gray-200 transition-colors"
                                        >
                                            Timestamp
                                            <span className="text-xs">{sortDir === 'asc' ? '↑' : '↓'}</span>
                                        </button>
                                    </th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Event Description</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Investigator Notes</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400">Source</th>
                                    <th className="pb-2 pr-4 font-medium text-gray-400 w-24"></th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-800">
                                {displayed.map((entry) => {
                                    if (editingId === entry.entry_id) {
                                        return (
                                            <tr key={entry.entry_id} className="bg-gray-800/60">
                                                <td className="py-2 pr-4 align-top" colSpan={5}>
                                                    <div className="space-y-2">
                                                        <div className="grid grid-cols-2 gap-2">
                                                            <div>
                                                                <label className={LABEL_CLASS}>Timestamp</label>
                                                                <input
                                                                    type="text"
                                                                    value={editForm.timestamp}
                                                                    onChange={(e) => setEditForm((f) => ({ ...f, timestamp: e.target.value }))}
                                                                    className={INPUT_CLASS}
                                                                />
                                                            </div>
                                                            <div>
                                                                <label className={LABEL_CLASS}>Display Timezone</label>
                                                                <TimezoneSelect
                                                                    value={editForm.displayTimezone}
                                                                    onChange={(v) => setEditForm((f) => ({ ...f, displayTimezone: v }))}
                                                                    className={INPUT_CLASS}
                                                                />
                                                            </div>
                                                        </div>
                                                        <div>
                                                            <label className={LABEL_CLASS}>Event Description</label>
                                                            <input
                                                                type="text"
                                                                value={editForm.eventDescription}
                                                                onChange={(e) => setEditForm((f) => ({ ...f, eventDescription: e.target.value }))}
                                                                className={INPUT_CLASS}
                                                            />
                                                        </div>
                                                        <div>
                                                            <label className={LABEL_CLASS}>Investigator Notes</label>
                                                            <textarea
                                                                rows={3}
                                                                value={editForm.investigatorNotes}
                                                                onChange={(e) => setEditForm((f) => ({ ...f, investigatorNotes: e.target.value }))}
                                                                className={INPUT_CLASS + ' resize-none'}
                                                            />
                                                        </div>
                                                        {editError && (
                                                            <p className="text-xs text-red-400">{editError}</p>
                                                        )}
                                                        <div className="flex gap-2">
                                                            <button
                                                                onClick={handleSaveEdit}
                                                                disabled={saving}
                                                                className="text-xs bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 text-white px-3 py-1 rounded transition-colors"
                                                            >
                                                                {saving ? 'Saving...' : 'Save'}
                                                            </button>
                                                            <button
                                                                onClick={() => setEditingId(null)}
                                                                className="text-xs text-gray-400 hover:text-gray-200 px-3 py-1 rounded border border-gray-600 hover:border-gray-400 transition-colors"
                                                            >
                                                                Cancel
                                                            </button>
                                                        </div>
                                                    </div>
                                                </td>
                                            </tr>
                                        );
                                    }

                                    if (confirmDelete === entry.entry_id) {
                                        return (
                                            <tr key={entry.entry_id} className="bg-red-900/20">
                                                <td colSpan={5} className="py-3 px-2">
                                                    <div className="flex items-center gap-3">
                                                        <span className="text-sm text-red-300">Delete this timeline entry? This cannot be undone.</span>
                                                        <button
                                                            onClick={() => handleDelete(entry.entry_id)}
                                                            className="text-xs bg-red-700 hover:bg-red-600 text-white px-3 py-1 rounded transition-colors"
                                                        >
                                                            Delete
                                                        </button>
                                                        <button
                                                            onClick={() => setConfirmDelete(null)}
                                                            className="text-xs text-gray-400 hover:text-gray-200 px-3 py-1 rounded border border-gray-600 hover:border-gray-400 transition-colors"
                                                        >
                                                            Cancel
                                                        </button>
                                                    </div>
                                                </td>
                                            </tr>
                                        );
                                    }

                                    const localTime = entry.display_timezone
                                        ? formatInTimezone(entry.timestamp, entry.display_timezone)
                                        : '';

                                    return (
                                        <tr key={entry.entry_id} className="hover:bg-gray-800/50">
                                            <td className="py-2 pr-4 align-top">
                                                <div className="font-mono text-xs text-gray-200 whitespace-nowrap">
                                                    {entry.timestamp}
                                                </div>
                                                {localTime && (
                                                    <div className="font-mono text-xs text-gray-500 mt-0.5 whitespace-nowrap">
                                                        {localTime}
                                                    </div>
                                                )}
                                            </td>
                                            <td className="py-2 pr-4 align-top text-gray-200 text-sm">
                                                {entry.event_description}
                                            </td>
                                            <td className="py-2 pr-4 align-top text-gray-400 text-sm whitespace-pre-wrap">
                                                {entry.investigator_notes}
                                            </td>
                                            <td className="py-2 pr-4 align-top text-xs">
                                                {entry.evidence_item_id ? (
                                                    <button
                                                        onClick={() => onNavigate?.(`evidence-notes-${entry.evidence_item_id}`)}
                                                        className="text-blue-400 hover:text-blue-300 underline transition-colors"
                                                    >
                                                        {evidenceLabel(entry.evidence_item_id, evidenceItems)}
                                                    </button>
                                                ) : (
                                                    <span className="text-gray-600">Case Level</span>
                                                )}
                                            </td>
                                            <td className="py-2 align-top">
                                                <div className="flex gap-3">
                                                    <button
                                                        onClick={() => startEdit(entry)}
                                                        className="text-xs text-gray-400 hover:text-blue-400 transition-colors"
                                                    >
                                                        Edit
                                                    </button>
                                                    <button
                                                        onClick={() => setConfirmDelete(entry.entry_id)}
                                                        className="text-xs text-gray-400 hover:text-red-400 transition-colors"
                                                    >
                                                        Delete
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </div>
    );
}
